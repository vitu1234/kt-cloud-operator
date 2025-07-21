package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	"dcnlab.ssu.ac.kr/kt-cloud-operator/internal/cloudapi"
	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// FetchAndCreateKubeconfigSecret waits for VM's HTTP server and creates kubeconfig secret
/*func FetchAndCreateKubeconfigSecret(k8sClient client.Client, machine *v1beta1.KTMachine, vmIP string) error {
	url := fmt.Sprintf("http://%s:8000/admin.conf", vmIP)
	var kubeconfigData []byte
	var fetchErr error

	// Retry mechanism: wait up to 2 minutes
	for i := 0; i < 24; i++ {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			kubeconfigData, fetchErr = io.ReadAll(resp.Body)
			resp.Body.Close()
			if fetchErr == nil {
				break
			}
		} else {
			if resp != nil {
				resp.Body.Close()
			}
		}
		time.Sleep(5 * time.Second)
	}

	if kubeconfigData == nil || fetchErr != nil {
		return fmt.Errorf("failed to fetch kubeconfig from %s: %w", url, fetchErr)
	}

	// Create kubeconfig secret
	secretName := machine.Name + "-kubeconfig"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "default",
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": machine.Name,
			},
			Annotations: map[string]string{
				"internal.kpt.dev/upstream-identifier": fmt.Sprintf("|Secret|default|%s", secretName),
				"nephio.org/cluster-name":              machine.Name,
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

	logger1.Infof("Kubeconfig secret '%s' created for %s", secretName, vmIP)
	return nil
}
*/
func FetchAndCreateKubeconfigSecret(k8sClient client.Client, machine *v1beta1.KTMachine) error {
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
	secretName := machine.Name + "-kubeconfig"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "default",
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": machine.Name,
			},
			Annotations: map[string]string{
				"internal.kpt.dev/upstream-identifier": fmt.Sprintf("|Secret|default|%s", secretName),
				"nephio.org/cluster-name":              machine.Name,
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
	return nil
}
