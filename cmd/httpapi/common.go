package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	v1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RequestPayloadDelete struct {
	ForceDelete string `json:"forceDelete"`
}

func DeleteVM(machine *v1beta1.KTMachine, token string) error {
	serverResponse, err := GetCreatedVM(machine, token)
	if err != nil {
		logger1.Error("Error creating GET VM request for deletion:", err)
		return err
	}

	//if serverResponse == nil  it means that the machine is already deleted
	if serverResponse == nil {
		logger1.Error("Machine already deleted: serverResponse is nil")
		return nil
	}

	//delete all the dependent resources
	err = DeleteVMDependentResources(machine, token)
	if err != nil {
		logger1.Error("Error deleting dependent resources:", err)
		return err
	}

	//actually delete the machine
	payload := RequestPayloadDelete{
		ForceDelete: "null",
	}

	// Marshal the payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger1.Error("Error marshaling JSON for POST machine deletion:", err)
		return err
	}

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/server/servers/" + machine.Status.ID + "/action"

	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger1.Error("Error creating POST DELETION VM request:", err)
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with your actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending delete machine request:", err)
		return err
	}
	defer resp.Body.Close()

	// Handle the response
	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return err
	}

	logger1.Info("-----------------------------------------")
	logger1.Info("Response Body on Delete machine:", string(body))
	logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and got machine!")
		// Parse the JSON into the struct
		// var serverResponse ServerResponse
		// err = json.Unmarshal(body, &serverResponse)
		// if err != nil {
		// 	logger1.Error("Error unmarshaling JSON response:", err)
		// 	return err
		// }

		return nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return errors.New("GET request failed with status: " + resp.Status)
	}

}

// get the machine
func GetCreatedVM(machine *v1beta1.KTMachine, token string) (*v1beta1.KTMachineStatus, error) {

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/server/servers/" + machine.Status.ID

	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", apiURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		logger1.Error("Error creating GET VM request:", err)
		return nil, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with your actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Handle the response
	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return nil, err
	}

	// logger1.Info("-----------------------------------------")
	// logger1.Info("Response Body:", string(body))
	// logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and got machine!")
		// Parse the JSON into the struct
		var serverResponse ServerResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return nil, err
		}
		// logger1.Error("Error unmarshaling JSON response:", err)
		return &serverResponse.Server, nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return nil, errors.New("GET request failed with status: " + resp.Status)
	}

}

// func ForceDeleteVM(machine *v1beta1.KTMachine, token string) error {
// 	serverResponse, err := GetCreatedVM(machine, token)
// 	if err != nil {
// 		logger1.Error("Error creating GET VM request for deletion:", err)
// 		return err
// 	}
// }

func DeleteVMDependentResources(machine *v1beta1.KTMachine, token string) error {
	if len(machine.Status.AssignedPublicIps) > 0 {
		return nil
	}

	err := DeleteFirewallSettings(machine, token)
	if err != nil {
		return err
	}

	for i := 0; i < len(machine.Status.AssignedPublicIps); i++ {
		//delete all the public ips assigned to the machine
		err := DeleteStaticNatOnCloud(machine.Status.AssignedPublicIps[i].StaticNatId, token)
		if err != nil {
			logger1.Error("Error deleting public IP:", err)
		}
	}

	return nil

}

func DeleteFirewallSettings(machine *v1beta1.KTMachine, token string) error {
	ctx := context.Background()
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

	existingFirewallSettings := &v1beta1.KTNetworkFirewall{}
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      machine.Name,
		Namespace: machine.Namespace,
	}, existingFirewallSettings)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Object does not exist, create it
			logger1.Info("KTNetworkFirewall does not exist, creating a new one")
			return err
		} else {
			// Error fetching object
			logger1.Errorf("Failed to fetch KTNetworkFirewall object: %v", err)
			return err
		}
	}

	jobIdsList := existingFirewallSettings.Status.FirewallJobs
	for i := 0; i < len(jobIdsList); i++ {
		err = DeleteFirewallOnCloud(jobIdsList[i].JobId, token)
		if err != nil {
			logger1.Error("Error deleting firewall:", err)
		}
	}
	err = k8sClient.Delete(ctx, existingFirewallSettings)
	if err != nil {
		logger1.Error("Error deleting firewall:", err)
		return err
	}
	return nil

}

func DeleteFirewallOnCloud(jobId, token string) error {
	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/Firewall/" + jobId

	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("DELETE", apiURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		logger1.Error("Error creating POST DELETION firewall request:", err)
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with your actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending delete firewall request:", err)
		return err
	}
	defer resp.Body.Close()

	// Handle the response
	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return err
	}

	logger1.Info("-----------------------------------------")
	logger1.Info("Response Body on Delete firewall:", string(body))
	logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and got firewall!")
		// Parse the JSON into the struct
		var serverResponse ServerResponse
		err = json.Unmarshal(body, &serverResponse)
		// if err != nil {
		// 	logger1.Error("Error unmarshaling JSON response:", err)
		// 	return err
		// }

		return nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return errors.New("GET request failed with status: " + resp.Status)
	}
}

func DeleteStaticNatOnCloud(staticNatId, token string) any {
	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/StaticNat/" + staticNatId

	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("DELETE", apiURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		logger1.Error("Error creating POST DELETION StaticNAT request:", err)
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with your actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending delete StaticNAT request:", err)
		return err
	}
	defer resp.Body.Close()

	// Handle the response
	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return err
	}

	logger1.Info("-----------------------------------------")
	logger1.Info("Response Body on Delete StaticNAT:", string(body))
	logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and got StaticNAT!")
		// Parse the JSON into the struct
		var serverResponse ServerResponse
		err = json.Unmarshal(body, &serverResponse)
		// if err != nil {
		// 	logger1.Error("Error unmarshaling JSON response:", err)
		// 	return err
		// }

		return nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return errors.New("GET request failed with status: " + resp.Status)
	}
}
