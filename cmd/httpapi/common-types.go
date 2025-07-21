package httpapi

type PublicNetwork struct {
	NcListentPublicIpsResponse NcListentPublicIpsResponse `json:"nc_listentpublicipsresponse"`
}

type NcListentPublicIpsResponse struct {
	HttpStatus int        `json:"httpStatus"`
	Data       []PublicIp `json:"data"`
}

type PublicIp struct {
	Account         string      `json:"accountId"`
	VPCId           string      `json:"vpcId"`
	ZoneId          string      `json:"zoneId"`
	ID              string      `json:"publicIpId"`
	Type            string      `json:"type"`
	CIDR            string      `json:"cidr"`
	IP              string      `json:"publicIp"`
	PortalZoneId    string      `json:"portalZoneId"`
	SerialIP        string      `json:"serialIp"`
	VLANId          int         `json:"vlanId"`
	PublicIpPoolId  string      `json:"publicIpPoolId"`
	IsAllocate      bool        `json:"isAllocate"`
	StaticNats      []StaticNat `json:"staticNats"`
	PortForwardings []any       `json:"portForwardings"` // use specific type if structure is known
}

type StaticNat struct {
	ID         string `json:"staticNatId"`
	Name       string `json:"name"`
	VPCId      string `json:"vpcId"`
	MappedIP   string `json:"mappedIp"`
	NetworkId  string `json:"networkId"`
	PublicIpId string `json:"publicIpId"`
	PublicIp   string `json:"publicIp"`
}

// Post Request Payload attach nat
type PostPayload struct {
	VMGuestIP     string `json:"vmguestip"`
	VMNetworkId   string `json:"vmnetworkid"`
	EntPublicIPId string `json:"entpublicipid"`
}

// POst payload for Firewall settings
type FirewallRuleRequest struct {
	SrcNat      bool     `json:"srcNat"`
	StartPort   string   `json:"startport"`
	EndPort     string   `json:"endport"`
	Protocol    string   `json:"protocol"`
	Action      string   `json:"action"` // "true" as string in JSON, not boolean
	SrcNetwork  []string `json:"srcNetwork"`
	DstNetwork  []string `json:"dstNetwork,omitempty"`
	StaticNatId string   `json:"staticNatId"`
	SrcAddress  []string `json:"srcAddress,omitempty"`
	DstAddress  []string `json:"dstAddress,omitempty"`
}

type MachinePrivateAddresses struct {
	NetworkName string `json:"networkname"`
	Address     string `json:"address"`
}

// attach NAT response
type NATAttachResponse struct {
	HttpStatus int                 `json:"httpStatus"`
	Data       StaticNatIdResponse `json:"data"`
}

type StaticNatIdResponse struct {
	StaticNatId string `json:"staticNatId"`
}

// create firewall settings response
type AddFirewallSettingsResponse struct {
	NcCreateFirewallRuleResponse NcCreateFirewallRuleResponse `json:"nc_createfirewallruleresponse"`
}

type NcCreateFirewallRuleResponse struct {
	DisplayText string `json:"displaytext"`
	Success     bool   `json:"success"`
	JobId       string `json:"job_id"`
}

// get networks response

type NcListOsNetworksResponse struct {
	Networks []NetworkData `json:"networks"`
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

// Response for getting networking Job Ids
type QueryAsyncJobResultResponse struct {
	HttpStatus int                `json:"httpStatus"`
	JobId      string             `json:"jobId"`
	Data       AsyncJobResultData `json:"data"`
	JobStatus  string             `json:"jobStatus"`
}
type AsyncJobResultData struct {
	JobId    *string `json:"jobId"`    // nullable, use pointer
	Detail   *string `json:"detail"`   // nullable
	VpcId    *string `json:"vpcId"`    // nullable
	PolicyId *string `json:"policyId"` // optional non-null value
}

type NcQueryAsyncJobResultResponse struct {
	Result Result `json:"result"`
	// State  string `json:"state"`
}

type Result struct {
	IPAddress   string `json:"ipaddress"`
	DisplayText string `json:"displaytext"`
	Success     bool   `json:"success"`
	ID          string `json:"id"`
}
