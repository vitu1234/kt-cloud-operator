package httpapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	"dcnlab.ssu.ac.kr/kt-cloud-operator/internal/cloudapi"
	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func ProcessEnvVariables() (cloudapi.Config, *zap.SugaredLogger) {
	var Config1 cloudapi.Config
	var logger2 *zap.SugaredLogger
	err := envconfig.Process("", &Config1)
	if err != nil {
		panic(err.Error())
	}
	err, logger2 = logger(Config1.LogLevel)
	if err != nil {
		panic(err.Error())
	}

	logger2.Info("Processed Env Variables...")
	return Config1, logger2
}

func logger(logLevel string) (error, *zap.SugaredLogger) {
	var level zapcore.Level
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		return err, nil
	}
	logConfig := zap.NewDevelopmentConfig()
	logConfig.Level.SetLevel(level)
	log, err := logConfig.Build()
	if err != nil {
		return err, nil
	}
	return nil, log.Sugar()
}

func getRestConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return ctrl.GetConfig()
}

// getClient initializes a controller-runtime Manager and returns the client it uses.
func getClient(config *rest.Config, scheme *runtime.Scheme) (client.Client, error) {
	// Register your custom resource's types
	if err := v1beta1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add custom resources to scheme: %v", err)
	}

	return client.New(config, client.Options{Scheme: scheme})
}

// we have to create a kubeconfig secret for the control plane
func FetchAndCreateKubeconfigSecret(k8sClient client.Client, machine *v1beta1.KTMachine, cluster *v1beta1.KTCluster) error {
	url := fmt.Sprintf("http://%s:8000/admin.conf", machine.Status.AssignedPublicIps[0].IP)

	client := &http.Client{Timeout: 30 * time.Second}

	var kubeconfigData []byte
	var fetchErr error

	// Retry mechanism: wait up to 2 minutes, check every 5 seconds
	for i := 0; i < 24; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			logger1.Error("Error creating GET kubeconfig request:", err)
			return err
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			logger1.Errorf("Error sending GET request to %s: %v", url, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			kubeconfigData, fetchErr = io.ReadAll(resp.Body)
			resp.Body.Close()
			if fetchErr == nil {
				logger1.Infof("Successfully fetched kubeconfig from %s", url)
				break
			} else {
				logger1.Errorf("Error reading response body: %v", fetchErr)
			}
		} else {
			logger1.Errorf("GET request to %s failed with status: %s", url, resp.Status)
			resp.Body.Close()
		}
		time.Sleep(5 * time.Second)
	}

	if kubeconfigData == nil || fetchErr != nil {
		return fmt.Errorf("failed to fetch kubeconfig from %s: %w", url, fetchErr)
	}

	// Create kubeconfig secret
	secretName := cluster.Name + "-kubeconfig"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "kt-cloud-operator-system",
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": cluster.Name,
			},
			Annotations: map[string]string{
				"internal.kpt.dev/upstream-identifier": fmt.Sprintf("|Secret|kt-cloud-operator-system|%s", secretName),
				"nephio.org/cluster-name":              cluster.Name,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"value": kubeconfigData,
		},
	}

	ctx := context.Background()
	err := k8sClient.Create(ctx, secret)
	if err != nil {
		return fmt.Errorf("failed to create kubeconfig secret: %w", err)
	}

	logger1.Infof("Kubeconfig secret '%s' created for %s", secretName, machine.Status.AssignedPublicIps[0].IP)

	changeKubeconfigDomain(secretName, "kt-cloud-operator-system", k8sClient, machine.Status.AssignedPublicIps[0].IP)
	return nil
}

func changeKubeconfigDomain(secretName, namespace string, clientset client.Client, newDomain string) error {

	logger1.Info("Changing kubeconfig server domain...")

	// Use in-cluster config
	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	return fmt.Errorf("failed to load in-cluster config: %v", err)
	// }

	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	return fmt.Errorf("failed to create clientset: %v", err)
	// }

	ctx := context.Background()
	secret := &corev1.Secret{}
	err := clientset.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
	if err != nil {
		return fmt.Errorf("failed to get secret: %v", err)
	}

	rawData, err := base64.StdEncoding.DecodeString(string(secret.Data["value"]))
	if err != nil {
		return fmt.Errorf("failed to decode kubeconfig: %v", err)
	}

	var kubeconfig clientcmdapi.Config
	if err := yaml.Unmarshal(rawData, &kubeconfig); err != nil {
		return fmt.Errorf("failed to unmarshal kubeconfig: %v", err)
	}

	// Update server domain
	for name, cluster := range kubeconfig.Clusters {
		u, err := url.Parse(cluster.Server)
		if err != nil {
			return fmt.Errorf("invalid server URL in cluster %s: %v", name, err)
		}
		u.Host = newDomain + ":" + strings.Split(u.Host, ":")[1]
		cluster.Server = u.String()
	}

	updatedData, err := yaml.Marshal(&kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to marshal updated kubeconfig: %v", err)
	}

	secret.Data["value"] = updatedData

	err = clientset.Update(ctx, secret)
	if err != nil {
		return fmt.Errorf("failed to update secret: %v", err)
	}

	fmt.Println("Successfully updated kubeconfig server domain.")
	return nil
}
