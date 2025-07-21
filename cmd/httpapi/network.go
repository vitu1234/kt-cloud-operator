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

	// Meta API for object metadata

	v1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AttachPublicIP(machine *v1beta1.KTMachine, token string) error {

	var machinePrivateAddresses []MachinePrivateAddresses
	// Iterate over dynamic keys in "addresses"
	for network, addresses := range machine.Status.Addresses {
		fmt.Printf("Network: %s\n", network)
		for _, addr := range addresses {
			fmt.Printf("  Address: %s\n", addr.Addr)
			fmt.Printf("  Version: %d\n", addr.Version)
			machineAddress := MachinePrivateAddresses{
				NetworkName: network,
				Address:     addr.Addr,
			}
			machinePrivateAddresses = append(machinePrivateAddresses, machineAddress)
		}
	}

	if len(machinePrivateAddresses) == 0 {
		return errors.New("failed to get machine address to pair with public ip address for firewall settings")
	}

	mappedIp := machinePrivateAddresses[0].Address        //just get the first IP address
	networkName := machinePrivateAddresses[0].NetworkName //just get the first IP address
	// vmnetworkid := machine.Spec.NetworkTier[0].ID         //just get the first tier

	// networkData, err := GetNetworkAllNetworks(token)

	// if err != nil {
	// 	return err
	// }
	// if len(networkData) == 0 {
	// 	return errors.New("failed to retrieve network data from cloud api call in the VPC, maybe try creating a network in the cloud in same zone as the cluster")
	// }

	// // find the network ID by looping through the networkData
	// var networkID string
	// for _, network := range networkData {

	// 	if network.Name == networkName {
	// 		networkID = network.ID
	// 		break
	// 	}
	// }
	// if networkID == "" {
	// 	return errors.New("failed to find network ID for network name: " + networkName)
	// }

	publicIPs, err := GetAvailablePublicIpAddresses(token)

	if err != nil {
		return err
	}
	if len(publicIPs) == 0 {
		return errors.New("no available public ip addresses on the cloud, maybe try creating in the cloud in same zone as the cluster")
	}
	publicIpId := publicIPs[0].ID

	networkAttachRequest := PostPayload{
		VMGuestIP:     mappedIp,
		EntPublicIPId: publicIpId,
	}

	// Marshal the struct to JSON
	payload, err := json.Marshal(networkAttachRequest)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	// Define the endpoint URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nsm/v1/staticNat"

	// Set up HTTP client with timeout
	// Set up the HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a new HTTP POST request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
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
	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return err
	}
	logger1.Info("Response Body:", string(body))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("POST request successful and attached Ip Address to machine!")

		// Parse the JSON into the struct
		var serverResponse NATAttachResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return err
		}

		if serverResponse.HttpStatus < 200 || serverResponse.HttpStatus >= 300 {
			return errors.New("Failed to attach public IP: " + serverResponse.Data.StaticNatId)
		}

		logger1.Info("NAT Attach Response Text: " + serverResponse.Data.StaticNatId)

		// logger1.Info("Didnt pass here")

		// Update the machine
		// Update the machine K8s Resource
		clientConfig, err := getRestConfig(Config.Kubeconfig)
		if err != nil {
			logger1.Errorf("Cannot prepare k8s client config: %v. Kubeconfig was: %s", err, Config.Kubeconfig)
			return err
		}
		// Set up a scheme (use runtime.Scheme from apimachinery)
		scheme := runtime.NewScheme()
		// Create Kubernetes client
		k8sClient, err := getClient(clientConfig, scheme)
		if err != nil {
			logger1.Fatalf("Failed to create Kubernetes client: %v", err)
			return err
		}
		machineStatusCopy := machine.Status
		assignedIp := v1beta1.AssignedPublicIps{
			Id:          publicIPs[0].ID,
			IP:          publicIPs[0].IP,
			StaticNatId: serverResponse.Data.StaticNatId,
			PairedPvtNetwork: v1beta1.PairedPvtNetwork{
				NetworkName: networkName,
				// NetworkID:   machinePrivateAddresses[0].NetworkName,
				// NetworkOsID: networkData.OSNetworkID,
				VMPvtIP: mappedIp,
			},
		}
		machineStatusCopy.AssignedPublicIps = append(machineStatusCopy.AssignedPublicIps, assignedIp)

		err = updateVMStatus(k8sClient, machine, &machineStatusCopy, machineStatusCopy.Status)
		if err != nil {
			logger1.Errorf("Failed to update VMstatus with public IP: %v", err)
			return err
		}
		logger1.Info("Updated the status of machine with public IP")
		return nil

	} else {
		logger1.Error("POST request failed with status:", resp.Status)
		return errors.New("post request failed with status:" + resp.Status)
	}
}

// get all unassigned public IPs
func GetAvailablePublicIpAddresses(token string) ([]PublicIp, error) {

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nsm/v1/publicIp"

	// Set up the HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger1.Error("Error creating GET request:", err)
		return nil, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and received public IPs!")

		var serverResponse NcListentPublicIpsResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return nil, err
		}

		filteredIps := []PublicIp{}
		for _, ip := range serverResponse.Data {
			if ip.Type == "ASSOCIATE" {
				filteredIps = append(filteredIps, ip)
			}
		}

		return filteredIps, nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return nil, errors.New("GET request failed with status: " + resp.Status)
	}
}

// get all assigned public IPs
func GetAssignedPublicIpAddresses(token string) ([]PublicIp, error) {

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/IpAddress"

	// Set up the HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger1.Error("Error creating GET request:", err)
		return nil, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and received public IPs!")

		var serverResponse NcListentPublicIpsResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return nil, err
		}

		filteredIps := []PublicIp{}
		for _, ip := range serverResponse.Data {
			if ip.Type == "STATICNAT" && len(ip.StaticNats) > 0 {
				filteredIps = append(filteredIps, ip)
			}
		}

		return filteredIps, nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return nil, errors.New("GET request failed with status: " + resp.Status)
	}
}

// get network
func GetNetworkIdByName(token, network_name string) (NetworkData, error) {
	apiURL := Config.ApiBaseURL + Config.Zone + "/nsm/network?networkType=ALL"
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger1.Error("Error creating GET request:", err)
		return NetworkData{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return NetworkData{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return NetworkData{}, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var serverResponse ListNetworksResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return NetworkData{}, err
		}

		for _, network := range serverResponse.Data {
			if network.Name == network_name {
				return network, nil
			}
		}

		return NetworkData{}, fmt.Errorf("network with name '%s' not found", network_name)

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return NetworkData{}, fmt.Errorf("GET request failed with status: %s", resp.Status)
	}
}

// get all networks in VPC
func GetNetworkAllNetworks(token string) ([]NetworkData, error) {
	apiURL := Config.ApiBaseURL + Config.Zone + "/nsm/network?networkType=ALL"
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger1.Error("Error creating GET request:", err)
		return []NetworkData{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return []NetworkData{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return []NetworkData{}, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var serverResponse ListNetworksResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return []NetworkData{}, err
		}

		return []NetworkData{}, fmt.Errorf("No networks found associated with any VPC")

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return []NetworkData{}, fmt.Errorf("GET request failed with status: %s", resp.Status)
	}
}

// get vpc networks
func GetListVpcNetworks(token string) ([]NetworkData, error) {

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/VPC"

	// Set up the HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", apiURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		logger1.Error("Error creating GET VM request:", err)
		return []NetworkData{}, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return []NetworkData{}, err
	}
	defer resp.Body.Close()

	// Handle the response
	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return []NetworkData{}, err
	}

	// logger1.Info("-----------------------------------------")
	// logger1.Info("Response Body Networks:", string(body))
	// logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and got machine!")
		// Parse the JSON into the struct
		var serverResponse ListVpcsResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return []NetworkData{}, err
		}

		filteredResponse := serverResponse.NcListVpcResponse.Vpcs[0].Networks //HOW TO IDENTIFY A VPC INCASE WE HAVE MULTIPLE VPCs

		return filteredResponse, nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return []NetworkData{}, errors.New("GET request failed with status: " + resp.Status)
	}

}

// add firewall settings
func AddFirewallSettings(machine *v1beta1.KTMachine, token string, securityGroupRules v1beta1.SecurityGroupRule, enableOutboundInternetTraffic bool) error {

	publicIPs, err := GetAssignedPublicIpAddresses(token)

	if err != nil {
		return err
	}
	if len(publicIPs) == 0 {
		return errors.New("no ip addresses have been assigned to find public address for this machine")
	}

	//get network id for the external network
	networkList, err := GetNetworkAllNetworks(token)
	if err != nil {
		return err
	}
	if len(networkList) == 0 {
		return errors.New("failed to get external network from cloud api call")
	}

	var externalNetworkID string
	for i := 0; i < len(networkList); i++ {
		if networkList[i].Name == "external" || networkList[i].Type == "UNTRUST" {
			externalNetworkID = networkList[i].ID
			break
		}
	}

	if len(externalNetworkID) == 0 {
		return errors.New("failed to find external network ID")
	}

	//for enabling outbound internet traffic
	// var from_internet string
	// var from_internal string
	var staticNatId string

	for z := 0; z < len(machine.Status.AssignedPublicIps); z++ {
		// get static nat id
		staticNatId = machine.Status.AssignedPublicIps[z].StaticNatId
	}

	firewallSettingsRequest := FirewallRuleRequest{
		SrcNat:    false,
		StartPort: securityGroupRules.StartPort,
		EndPort:   securityGroupRules.EndPort,

		Action:      securityGroupRules.Action,
		Protocol:    securityGroupRules.Protocol,
		SrcNetwork:  []string{externalNetworkID},
		StaticNatId: staticNatId,
	}

	// Marshal the struct to JSON
	payload, err := json.Marshal(firewallSettingsRequest)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	// Define the endpoint URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nsm/v1/firewall/policy"

	// Set up HTTP client with timeout
	// Set up the HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a new HTTP POST request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
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
	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return err
	}
	logger1.Info("Response Body:", string(body))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("POST request successful and added firewall settings for the cluster!")

		// Parse the JSON into the struct
		var serverResponse AddFirewallSettingsResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return err
		}

		responseText, _ := json.Marshal(serverResponse)
		logger1.Info("Response Text: " + string(responseText))

		if serverResponse.NcCreateFirewallRuleResponse.DisplayText != "" {
			return errors.New(serverResponse.NcCreateFirewallRuleResponse.DisplayText)
		}
		logger1.Info("Add firewall settings to the cluster ")

		groupRules := v1beta1.FirewallRules{
			SrcNat:    false,
			StartPort: securityGroupRules.StartPort,
			EndPort:   securityGroupRules.EndPort,

			Action:      securityGroupRules.Action,
			Protocol:    securityGroupRules.Protocol,
			SrcNetwork:  []string{externalNetworkID},
			StaticNatId: staticNatId,
		}

		logger1.Info("Firewall responce job id: ", serverResponse.NcCreateFirewallRuleResponse.JobId)

		//get the firewall id and create a firewall object in k8s
		rule_Id, err := GetNetworkingJobId(token, serverResponse.NcCreateFirewallRuleResponse.JobId, "Firewall_Create")
		if err != nil {
			logger1.Errorf("Failed to get job id: %v", err)
			return err
		}

		if rule_Id == "" {
			logger1.Errorf("Failed to get job id")
			return errors.New("failed to get job id for firewall settings")
		}

		err = createFirewallObjectInK8s(machine, groupRules, serverResponse.NcCreateFirewallRuleResponse.JobId, rule_Id)
		if err != nil {
			logger1.Errorf("Failed to create firewall object in k8s: %v", err)
			return err
		}

		return nil

	} else {
		logger1.Error("POST request failed with status:", resp.Status)
		return errors.New("post request failed with status:" + resp.Status)
	}
}

func createFirewallObjectInK8s(machine *v1beta1.KTMachine, securityGroupRules v1beta1.FirewallRules, s, rule_Id string) error {
	// panic("unimplemented")

	//check if the firewall object already exists
	// if it exists, update the object
	// if it does not exist, create the object
	// if it exists and the job id is the same, do nothing
	// if it exists and the job id is different, update the object
	logger1.Info("Creating Firewall object in K8s with job id: ", s)

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

	// ktFirewallRules := v1beta1.FirewallRules{
	// 	StartPort:    securityGroupRules.StartPort,
	// 	Protocol:     securityGroupRules.Protocol,
	// 	VirtualIPID:  securityGroupRules.VirtualIPID,
	// 	Action:       securityGroupRules.Action,
	// 	SrcNetworkID: securityGroupRules.SrcNetworkID,
	// 	DstIP:        securityGroupRules.DstIP,
	// 	EndPort:      securityGroupRules.EndPort,
	// 	DstNetworkID: securityGroupRules.DstNetworkID,
	// }

	ktFirewallJobs := v1beta1.FirewallJobs{
		JobId:     s,
		RuleId:    rule_Id,
		CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000000Z"),
	}

	existingFirewallObj := &v1beta1.KTNetworkFirewall{}
	err = k8sClient.Get(ctx, client.ObjectKey{Name: machine.Name, Namespace: machine.Namespace}, existingFirewallObj)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Object does not exist, create it
			logger1.Info("Firewall does not exist, creating a new one")

			firewall := &v1beta1.KTNetworkFirewall{
				ObjectMeta: metav1.ObjectMeta{
					Name:      machine.Name,
					Namespace: machine.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         machine.APIVersion,
							Kind:               machine.Kind,
							Name:               machine.Name,
							UID:                machine.UID,
							Controller:         pointer.Bool(true), // Indicates this is the managing controller
							BlockOwnerDeletion: pointer.Bool(true), // Prevent deletion of the machine until the firewall is deleted
						},
					},
				},
				Spec: v1beta1.KTNetworkFirewallSpec{
					FirewallRules: []v1beta1.FirewallRules{securityGroupRules},
				},
				Status: v1beta1.KTNetworkFirewallStatus{
					FirewallJobs: []v1beta1.FirewallJobs{ktFirewallJobs},
				},
			}
			err = k8sClient.Create(ctx, firewall)
			if err != nil {
				logger1.Errorf("Failed to create KTNetworkFirewall object: %v", err)
				return err
			}
			existingFirewallObj := &v1beta1.KTNetworkFirewall{}
			err = k8sClient.Get(ctx, client.ObjectKey{Name: machine.Name, Namespace: machine.Namespace}, existingFirewallObj)
			if err != nil {
				logger1.Errorf("Failed to fetch just created KTNetworkFirewall object to update its status: %v", err)
				return err
			}
			existingFirewallObj.Status.FirewallJobs = append(existingFirewallObj.Status.FirewallJobs, ktFirewallJobs)
			err = k8sClient.Status().Update(ctx, existingFirewallObj)
			if err != nil {
				logger1.Errorf("Failed to update KTNetworkFirewall object after just creating it: %v", err)
				return err
			}

			logger1.Info("KTNetworkFirewall object created successfully!")
			return nil
		} else {
			// Error fetching object
			logger1.Errorf("Failed to fetch KTNetworkFirewall object: %v", err)
			return err
		}
	} else {
		// Object exists, update it
		logger1.Info("KTNetworkFirewall already exists, updating it")
		existingFirewallObj.Spec.FirewallRules = append(existingFirewallObj.Spec.FirewallRules, securityGroupRules)
		existingFirewallObj.Status.FirewallJobs = append(existingFirewallObj.Status.FirewallJobs, ktFirewallJobs)
		err = k8sClient.Status().Update(ctx, existingFirewallObj)
		if err != nil {
			logger1.Errorf("Failed to update KTNetworkFirewall object: %v", err)
			return err
		}
		logger1.Info("KTNetworkFirewall object updated successfully!")
	}

	logger1.Info("Firewall object created successfully!")
	return nil
}

// get Job ID is for a "Firewall: Create" request,
func GetNetworkingJobId(token, job_id, job_type string) (string, error) {

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "nsm/v1/job/status/" + job_id

	// Set up the HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", apiURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		logger1.Error("Error GET Networking job id on cloud API request:", err)
		return "", err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error POST Networking job id sending request:", err)
		return "", err
	}
	defer resp.Body.Close()

	// Handle the response
	fmt.Println("Response POST Networking job id Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error POST Networking job id reading response body:", err)
		return "", err
	}

	logger1.Info("-----------------------------------------")
	logger1.Info("Response POST Networking job id Body Networks:", string(body))
	logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request POST Networking job id successful and got network job id!")
		// Parse the JSON into the struct
		var serverResponse QueryAsyncJobResultResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return "", err
		}

		logger1.Info("Response Text for firewall: " + string(body))

		if job_type == "Firewall_Create" {
			if serverResponse.Data.PolicyId != nil {
				return *serverResponse.Data.PolicyId, nil
			}
			return "", errors.New("PolicyId is nil")
		} else {
			if serverResponse.Data.VpcId != nil {
				return *serverResponse.Data.VpcId, nil
			}
			return "", errors.New("VpcId is nil")
		}

		// logger1.Info("Lenmhgth: ", serverResponse)
		// return filteredResponse, nil

	} else {
		logger1.Error("GET request POST Networking job id failed with status:", resp.Status)
		return "", errors.New("get request failed with status: " + resp.Status)
	}

}
