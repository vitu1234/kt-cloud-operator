package httpapi

// get networks response
type ListNetworksResponse struct {
	HttpStatus int           `json:"httpStatus"`
	Data       []NetworkData `json:"data"`
}

// data inside ListNetworksResponse
// NetworkData represents the structure of network data in the response
type NetworkData struct {
	Account       string `json:"accountId"`
	ID            string `json:"networkId"`
	ZoneID        string `json:"zoneId"`
	Type          string `json:"type"`
	VLAN          string `json:"interface"` // originally vlanId as int, but mapped from string "interface"
	VPCID         string `json:"vpcId"`
	ProjectID     string `json:"projectId"`
	Gateway       string `json:"gatewayIp"`
	CIDR          string `json:"cidr"`
	StartIP       string `json:"startIp"`
	EndIP         string `json:"endIp"`
	ISCSIStartIP  string `json:"iscsiStartIp"`
	ISCSIEndIP    string `json:"iscsiEndIp"`
	Name          string `json:"networkName"`
	Netmask       string `json:"netmaskIp"`
	Status        string `json:"status"`
	MainNetworkID string // leave empty if not in response
	DataLakeYN    string // leave empty if not in response
	OSNetworkID   string // leave empty if not in response
	SSLVPN        string // leave empty if not in response
}
