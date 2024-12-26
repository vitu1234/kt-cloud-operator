package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	// Meta API for object metadata

	v1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

type PublicNetwork struct {
	NcListentPublicIpsResponse NcListentPublicIpsResponse `json:"nc_listentpublicipsresponse"`
}

type NcListentPublicIpsResponse struct {
	PublicIps []PublicIp `json:"publicips"`
}

type PublicIp struct {
	EntPublicCIDRId string      `json:"entpubliccidrid"`
	VirtualIps      []VirtualIp `json:"virtualips"`
	VPCId           string      `json:"vpcid"`
	IP              string      `json:"ip"`
	ZoneId          string      `json:"zoneid"`
	Id              string      `json:"id"`
	Type            string      `json:"type"`
	Account         string      `json:"account"`
}

type VirtualIp struct {
	VMGuestIP   string `json:"vmguestip"`
	IPAddress   string `json:"ipaddress"`
	VPCId       string `json:"vpcid"`
	IPAddressId string `json:"ipaddressid"`
	Name        string `json:"name"`
	NetworkId   string `json:"networkid"`
	Id          string `json:"id"`
}

// Post Request Payload attach nat
type PostPayload struct {
	VMGuestIP     string `json:"vmguestip"`
	VMNetworkId   string `json:"vmnetworkid"`
	EntPublicIPId string `json:"entpublicipid"`
}

// POst payload for Firewall settings
type PostPayloadFirewallSettings struct {
	StartPort    int    `json:"startport"`
	EndPort      int    `json:"endport"`
	Action       string `json:"action"`
	Protocol     string `json:"protocol"`
	DstIp        string `json:"dstip"`
	VirtualIPId  string `json:"virtualipid"`
	SrcNetworkId string `json:"srcnetworkid"`
	DstNetworkId string `json:"dstnetworkid"`
}

type MachinePrivateAddresses struct {
	NetworkName string `json:"networkname"`
	Address     string `json:"address"`
}

// attach NAT response
type NATAttachResponse struct {
	NcEnableStaticNatResponse NcEnableStaticNatResponse `json:"nc_enablestaticnatresponse"`
}

type NcEnableStaticNatResponse struct {
	DisplayText string `json:"displaytext"`
	Success     bool   `json:"success"`
}

// get networks response
type ListNetworksResponse struct {
	NcListOsNetworksResponse NcListOsNetworksResponse `json:"nc_listosnetworksresponse"`
}
type NcListOsNetworksResponse struct {
	Networks []NetworkData `json:"networks"`
}

type NetworkData struct {
	EndIP   string `json:"endip"`
	Shared  string `json:"shared"`
	StartIP string `json:"startip"`
	Type    string `json:"type"`
	SSLVPN  string `json:"sslvpn"`
	VLAN    string `json:"vlan"`
	// EntPublicCIDRs []string `json:"entpubliccidrs"`
	Netmask       string `json:"netmask"`
	VPCID         string `json:"vpcid"`
	Name          string `json:"name"`
	MainNetworkID string `json:"mainnetworkid"`
	ZoneID        string `json:"zoneid"`
	DataLakeYN    string `json:"datalakeyn"`
	CIDR          string `json:"cidr"`
	ID            string `json:"id"`
	ProjectID     string `json:"projectid"`
	Gateway       string `json:"gateway"`
	ISCSIStartIP  string `json:"iscsistartip"`
	ISCSIEndIP    string `json:"iscsiendip"`
	Account       string `json:"account"`
	OSNetworkID   string `json:"osnetworkid"`
	Status        string `json:"status"`
}

type ListVpcsResponse struct {
	NcListVpcResponse NcListVpcResponse `json:"nc_listvpcsresponse"`
}
type NcListVpcResponse struct {
	Vpcs []Vpc `json:"vpcs"`
}

type Vpc struct {
	Networks []NetworkData `json:"networks"`
}

// list VPC response
type NcListVPCResponse struct {
	Networks []NetworkData `json:"networks"`
}

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

	vmguestip := machinePrivateAddresses[0].Address       //just get the first IP address
	networkName := machinePrivateAddresses[0].NetworkName //just get the first IP address
	vmnetworkid := machine.Spec.NetworkTier[0].ID         //just get the first tier

	networkData, err := GetNetworkIdByName(token, networkName)

	if err != nil {
		return err
	}
	if networkData.ID == "" {
		return errors.New("failed to retrieve network data by network name")
	}

	publicIPs, err := GetAvailablePublicIpAddresses(token)

	if err != nil {
		return err
	}
	if len(publicIPs.PublicIps) == 0 {
		return errors.New("no available public ip addresses on the cloud, maybe try creating in the cloud in same zone as the cluster")
	}
	entpublicipid := publicIPs.PublicIps[0].Id

	networkAttachRequest := PostPayload{
		VMGuestIP:     vmguestip,
		VMNetworkId:   vmnetworkid,
		EntPublicIPId: entpublicipid,
	}

	// Marshal the struct to JSON
	payload, err := json.Marshal(networkAttachRequest)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	// Define the endpoint URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/StaticNat"

	// Set up HTTP client with timeout
	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

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

		// logger1.Info("Response Text: " + serverResponse.NcEnableStaticNatResponse.DisplayText)

		if !serverResponse.NcEnableStaticNatResponse.Success {
			return errors.New(serverResponse.NcEnableStaticNatResponse.DisplayText)
		}

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
			Id: publicIPs.PublicIps[0].Id,
			IP: publicIPs.PublicIps[0].IP,
			PairedPvtNetwork: v1beta1.PairedPvtNetwork{
				NetworkName: networkName,
				NetworkID:   networkData.ID,
				NetworkOsID: networkData.OSNetworkID,
				VMPvtIP:     vmguestip,
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
func GetAvailablePublicIpAddresses(token string) (NcListentPublicIpsResponse, error) {

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/IpAddress"

	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", apiURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		logger1.Error("Error creating GET VM request:", err)
		return NcListentPublicIpsResponse{}, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return NcListentPublicIpsResponse{}, err
	}
	defer resp.Body.Close()

	// Handle the response
	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return NcListentPublicIpsResponse{}, err
	}

	// logger1.Info("-----------------------------------------")
	// logger1.Info("Response Body Networks:", string(body))
	// logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and got machine!")
		// Parse the JSON into the struct
		var serverResponse PublicNetwork
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return NcListentPublicIpsResponse{}, err
		}

		filteredResponse := NcListentPublicIpsResponse{}
		filteredPublicIps := []PublicIp{}
		for i := 0; i < len(serverResponse.NcListentPublicIpsResponse.PublicIps); i++ {
			publicIps := serverResponse.NcListentPublicIpsResponse.PublicIps
			if len(publicIps[i].VirtualIps) == 0 && publicIps[i].Type == "ASSOCIATE" {
				publicIP := PublicIp{
					EntPublicCIDRId: publicIps[i].EntPublicCIDRId,
					VirtualIps:      publicIps[i].VirtualIps,
					VPCId:           publicIps[i].VPCId,
					IP:              publicIps[i].IP,
					ZoneId:          publicIps[i].ZoneId,
					Type:            publicIps[i].Type,
					Id:              publicIps[i].Id,
					Account:         publicIps[i].Account,
				}
				filteredPublicIps = append(filteredPublicIps, publicIP)
			}
		}
		filteredResponse.PublicIps = filteredPublicIps

		return filteredResponse, nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return NcListentPublicIpsResponse{}, errors.New("GET request failed with status: " + resp.Status)
	}

}

// get all assigned public IPs
func GetAssignedPublicIpAddresses(token string) (NcListentPublicIpsResponse, error) {

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/IpAddress"

	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", apiURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		logger1.Error("Error creating GET VM request:", err)
		return NcListentPublicIpsResponse{}, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return NcListentPublicIpsResponse{}, err
	}
	defer resp.Body.Close()

	// Handle the response
	// fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return NcListentPublicIpsResponse{}, err
	}

	// logger1.Info("-----------------------------------------")
	// logger1.Info("Response Body Networks:", string(body))
	// logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and got machine!")
		// Parse the JSON into the struct
		var serverResponse PublicNetwork
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return NcListentPublicIpsResponse{}, err
		}

		filteredResponse := NcListentPublicIpsResponse{}
		filteredPublicIps := []PublicIp{}

		publicIps := serverResponse.NcListentPublicIpsResponse.PublicIps

		for i := 0; i < len(serverResponse.NcListentPublicIpsResponse.PublicIps); i++ {

			if len(publicIps[i].VirtualIps) > 0 && publicIps[i].Type == "STATICNAT" {

				// logger1.Info("For loop "+strconv.Itoa(i), publicIps)

				publicIP := PublicIp{
					EntPublicCIDRId: publicIps[i].EntPublicCIDRId,
					VirtualIps:      publicIps[i].VirtualIps,
					VPCId:           publicIps[i].VPCId,
					IP:              publicIps[i].IP,
					ZoneId:          publicIps[i].ZoneId,
					Type:            publicIps[i].Type,
					Id:              publicIps[i].Id,
					Account:         publicIps[i].Account,
				}
				filteredPublicIps = append(filteredPublicIps, publicIP)
			}
		}
		filteredResponse.PublicIps = filteredPublicIps

		return filteredResponse, nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return NcListentPublicIpsResponse{}, errors.New("GET request failed with status: " + resp.Status)
	}

}

// get network
func GetNetworkIdByName(token, network_name string) (NetworkData, error) {

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/Network"

	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", apiURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		logger1.Error("Error creating GET VM request:", err)
		return NetworkData{}, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token) // Replace with actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return NetworkData{}, err
	}
	defer resp.Body.Close()

	// Handle the response
	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger1.Error("Error reading response body:", err)
		return NetworkData{}, err
	}

	// logger1.Info("-----------------------------------------")
	// logger1.Info("Response Body Networks:", string(body))
	// logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("GET request successful and got machine!")
		// Parse the JSON into the struct
		var serverResponse ListNetworksResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return NetworkData{}, err
		}

		filteredResponse := NetworkData{}
		for i := 0; i < len(serverResponse.NcListOsNetworksResponse.Networks); i++ {
			// logger1.Info("NETWORK NAME SERVER: ", network_name)
			// logger1.Info("-------------------------------------")
			// logger1.Info("NETWORK 	FILTERED RESPONSE: ", serverResponse.NcListOsNetworksResponse.Networks[i].Name)
			if serverResponse.NcListOsNetworksResponse.Networks[i].Name == network_name {
				filteredResponse = serverResponse.NcListOsNetworksResponse.Networks[i]
			}
		}

		// logger1.Info("Lenmhgth: ", serverResponse)
		return filteredResponse, nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return NetworkData{}, errors.New("GET request failed with status: " + resp.Status)
	}

}

// get vpc networks
func GetListVpcNetworks(token string) ([]NetworkData, error) {

	// Define the API URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/VPC"

	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

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
func AddFirewallSettings(machine *v1beta1.KTMachine, token string, securityGroupRules v1beta1.SecurityGroupRule) error {

	publicIPs, err := GetAssignedPublicIpAddresses(token)

	if err != nil {
		return err
	}
	if len(publicIPs.PublicIps) == 0 {
		return errors.New("no ip addresses have been assigned to find public address for this machine")
	}

	//get network id
	vpcNetworks, err := GetListVpcNetworks(token)

	if err != nil {
		return err
	}

	if len(vpcNetworks) == 0 {
		return errors.New("failed to get vpc networks from cloud api call")
	}

	var virtualipid string
	var dstnetworkid string
	var srcnetworkid string

	for i := 0; i < len(publicIPs.PublicIps); i++ {
		if len(publicIPs.PublicIps[i].VirtualIps) > 0 {
			for y := 0; y < len(publicIPs.PublicIps[i].VirtualIps); y++ {
				for z := 0; z < len(machine.Status.AssignedPublicIps); z++ {
					if publicIPs.PublicIps[i].VirtualIps[y].IPAddress == machine.Status.AssignedPublicIps[z].IP {
						virtualipid = publicIPs.PublicIps[i].VirtualIps[y].Id
						if securityGroupRules.Direction == "ingress" {
							dstnetworkid = machine.Status.AssignedPublicIps[z].PairedPvtNetwork.NetworkID
							for i := 0; i < len(vpcNetworks); i++ {
								if vpcNetworks[i].Type == "PUBLIC" {
									srcnetworkid = vpcNetworks[i].ID
								}
							}
						} else {
							srcnetworkid = machine.Status.AssignedPublicIps[z].PairedPvtNetwork.NetworkID
							for i := 0; i < len(vpcNetworks); i++ {
								if vpcNetworks[i].Type == "PUBLIC" {
									dstnetworkid = vpcNetworks[i].ID
								}
							}
						}

						break
					}
				}
			}
		}
	}

	firewallSettingsRequest := PostPayloadFirewallSettings{
		StartPort:    securityGroupRules.StartPort,
		EndPort:      securityGroupRules.EndPort,
		Action:       securityGroupRules.Action,
		Protocol:     securityGroupRules.Protocol,
		DstIp:        securityGroupRules.Dstip,
		VirtualIPId:  virtualipid,
		SrcNetworkId: srcnetworkid,
		DstNetworkId: dstnetworkid,
	}

	// Marshal the struct to JSON
	payload, err := json.Marshal(firewallSettingsRequest)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	// Define the endpoint URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/nc/Firewall"

	// Set up HTTP client with timeout
	// Set up the HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

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
		var serverResponse NATAttachResponse
		err = json.Unmarshal(body, &serverResponse)
		if err != nil {
			logger1.Error("Error unmarshaling JSON response:", err)
			return err
		}

		// logger1.Info("Response Text: " + serverResponse.NcEnableStaticNatResponse.DisplayText)

		if !serverResponse.NcEnableStaticNatResponse.Success {
			return errors.New(serverResponse.NcEnableStaticNatResponse.DisplayText)
		}

		logger1.Info("Added firewall settings to the cluster ")
		return nil

	} else {
		logger1.Error("POST request failed with status:", resp.Status)
		return errors.New("post request failed with status:" + resp.Status)
	}
}
