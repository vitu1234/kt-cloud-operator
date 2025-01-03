package httpapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	v1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

func JoinControlPlane(ktMachineControlPlaneList []v1beta1.KTMachine, ktMachine v1beta1.KTMachine, token string) error {

	var controlPlaneMachine v1beta1.KTMachine
	for i := 0; i < len(ktMachineControlPlaneList); i++ {
		err := CheckControlPlaneMachineReady(&ktMachineControlPlaneList[i])
		if err == nil {
			controlPlaneMachine = ktMachineControlPlaneList[i]
			break
		}
		logger1.Info("Control Plane not ready yet for " + ktMachineControlPlaneList[i].Name + ", checking another control-plane machine")
	}

	if len(ktMachineControlPlaneList) == 0 {
		logger1.Info("ready and boostrapped control plane list is zero")
		return errors.New("ready and boostrapped control plane list is zero")
	}

	// we have to join the control plane
	err := CreateWorkerVM(&ktMachine, token, controlPlaneMachine.Status.AssignedPublicIps[0].PairedPvtNetwork.VMPvtIP) //just choose the first IP address paired

	if err != nil {
		return err
	}

	return nil

}

func CreateWorkerVM(machine *v1beta1.KTMachine, token, controlPlaneIP string) error {
	// Create the payload
	networks := []NetworkTier{}
	block_device_mapping_v2 := []BlockDeviceMappingV2{}

	for i, network := range machine.Spec.NetworkTier {
		fmt.Println(network.ID, i)
		networks = append(
			networks,
			NetworkTier{
				UUID: network.ID,
			})
	}

	for i, block_device_mapping := range machine.Spec.BlockDeviceMapping {
		fmt.Println(block_device_mapping.ID, i)
		block_device_mapping_v2 = append(
			block_device_mapping_v2,
			BlockDeviceMappingV2{
				UUID:            block_device_mapping.ID,
				BootIndex:       block_device_mapping.BootIndex,
				VolumeSize:      block_device_mapping.VolumeSize,
				SourceType:      block_device_mapping.SourceType,
				DestinationType: block_device_mapping.DestinationType,
			})
	}

	// Cloud-init configuration
	cloudInit := `#cloud-config
runcmd:
  - export K8S_API=$(echo "` + controlPlaneIP + `")
  - export DEFAULT_PORT=${DEFAULT_PORT:-8000}
  - sudo swapoff -a && sudo sed -i '/\bswap\b/d' /etc/fstab && [ -f /swap.img ] && sudo swapoff /swap.img
  - count=0
  - |
    while :; do
      if DATA=$(curl -s "$K8S_API:$DEFAULT_PORT/k8s"); then
        export K8S_API=$(echo "$DATA" | cut -d' ' -f1)
        export CAHASH=$(echo "$DATA" | cut -d' ' -f2)
        export TOKEN=$(echo "$DATA" | cut -d' ' -f3)
        break
      fi
      ((++count == 3600)) && break
      sleep 1
    done
  - |
    if [ -z "$K8S_API" ] || [ -z "$CAHASH" ] || [ -z "$TOKEN" ]; then
      echo "empty value"; exit 0
    fi
  - |
      sudo kubeadm join $K8S_API:6443 --token $TOKEN --discovery-token-ca-cert-hash sha256:$CAHASH`

	// Encode the cloud-init configuration in Base64
	encoded_user_data := base64.StdEncoding.EncodeToString([]byte(cloudInit))

	payload := RequestPayload{
		Server: Server{
			Name:                 machine.Name,
			KeyName:              machine.Spec.SSHKeyName,
			FlavorRef:            machine.Spec.Flavor,
			AvailabilityZone:     machine.Spec.AvailabilityZone,
			Networks:             networks,
			BlockDeviceMappingV2: block_device_mapping_v2,
			UserData:             encoded_user_data,
		},
	}

	// Marshal the payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger1.Error("Error marshaling JSON for POST worker machine creation:", err)
		return err
	}

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/server/servers"

	// Set up the HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a new HTTP POST request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger1.Error("Error creating request:", err)
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with your actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// Handle the response
	// fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return err
	}
	// logger1.Info("Response Body:", string(body))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("POST request successful and created worker machine!")

		// Parse the JSON into the struct
		var serverResponse ServerResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return err
		}

		// Access the parsed data
		// fmt.Println("Parsed Response:")
		// fmt.Printf("Server ID: %s\n", serverResponse.Server.ID)
		// fmt.Printf("Admin Password: %s\n", serverResponse.Server.AdminPass)
		// fmt.Printf("Disk Config: %s\n", serverResponse.Server.DiskConfig)
		// fmt.Println("Links:")
		// for _, link := range serverResponse.Server.Links {
		// 	fmt.Printf("  - Rel: %s, Href: %s\n", link.Rel, link.Href)
		// }
		// fmt.Println("Security Groups:")
		// for _, group := range serverResponse.Server.SecurityGroups {
		// 	fmt.Printf("  - Name: %s\n", group.Name)
		// }

		// Update the machine K8s Resource
		clientConfig, err := getRestConfig(Config.Kubeconfig)
		if err != nil {
			logger1.Errorf("Cannot prepare k8s client config: %v. Kubeconfig was: %s", err, Config.Kubeconfig)
			panic(err.Error())
		}
		// Set up a scheme (use runtime.Scheme from apimachinery)
		scheme := runtime.NewScheme()
		// Create Kubernetes client
		k8sClient, err := getClient(clientConfig, scheme)
		if err != nil {
			logger1.Fatalf("Failed to create Kubernetes client: %v", err)
			return err
		}

		// machineStatus := &v1beta1.KTMachineStatus{
		// 	ID: serverResponse.Server.ID,
		// 	AdminPass: serverResponse.Server.AdminPass,
		// 	Links: serverResponse.Server.Links,
		// }
		err = updateVMStatus(k8sClient, machine, &serverResponse.Server, "Creating")
		if err != nil {
			logger1.Errorf("Failed to update VMstatus: %v", err)
			return err
		}
		logger1.Info("Updated the status of machine")
		return nil

	} else {
		logger1.Error("POST request failed with status:", resp.Status)
	}

	return nil
}
