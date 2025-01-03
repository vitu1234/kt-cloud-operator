/*
Copyright 2024. DCN

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// KTMachineFinalizer allows ReconcileKTMachine to clean up resources associated with KTVM before
	// removing it from the apiserver.
	KTMachineFinalizer = "ktmachine.infrastructure.dcnlab.ssu.ac.kr"
)

// KTMachineSpec defines the desired state of KTMachine.
type KTMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KTMachine. Edit ktmachine_types.go to remove/update
	ClusterName        string               `json:"clusterName,omitempty"`
	Flavor             string               `json:"flavor,omitempty"`
	SSHKeyName         string               `json:"sshKeyName,omitempty"`
	BlockDeviceMapping []BlockDeviceMapping `json:"blockDeviceMapping,omitempty"`
	NetworkTier        []NetworkTier        `json:"networkTier,omitempty"`
	Networks           []Networks           `json:"networks,omitempty"`
	Ports              []Port               `json:"ports,omitempty"`
	AvailabilityZone   string               `json:"availabilityZone,omitempty"`
	UserData           string               `json:"userData,omitempty"`
	ControlPlaneNumber int                  `json:"controlPlaneNumber,omitempty"` //if machine is controlPlane we have to identify how many there are in order to join some controlPlanes to existing ones. Number 1 is the main and first one
}

type Networks struct {
	ID string `json:"id,omitempty"`
}

type AssignedPublicIps struct {
	IP               string           `json:"ip,omitempty"`
	Id               string           `json:"id,omitempty"`
	StaticNatId      string           `json:"staticNatId,omitempty"`
	PairedPvtNetwork PairedPvtNetwork `json:"pairedPvtNetwork,omitempty"`
}

type PairedPvtNetwork struct {
	NetworkName string `json:"networkName,omitempty"`
	NetworkID   string `json:"networkID,omitempty"`
	NetworkOsID string `json:"networkOsID,omitempty"`
	VMPvtIP     string `json:"vmPvtIp,omitempty"`
}

// KTMachineStatus defines the observed state of KTMachine.
type KTMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ID             string           `json:"id,omitempty"`
	AdminPass      string           `json:"adminPass,omitempty"`
	Links          []Links          `json:"links,omitempty"`
	SecurityGroups []SecurityGroups `json:"securityGroups,omitempty"`

	AssignedPublicIps []AssignedPublicIps `json:"AssignedPublicIPs,omitempty"`

	// New fields
	TenantID string `json:"tenant_id,omitempty"`
	// Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Addresses         map[string][]Address `json:"addresses,omitempty"`
	TaskState         *string              `json:"OS-EXT-STS:task_state,omitempty"`
	Description       *string              `json:"description,omitempty"`
	DiskConfig        string               `json:"OS-DCF:diskConfig,omitempty"`
	TrustedImageCerts *string              `json:"trusted_image_certificates,omitempty"`
	AvailabilityZone  string               `json:"OS-EXT-AZ:availability_zone,omitempty"`
	PowerState        int                  `json:"OS-EXT-STS:power_state,omitempty"`
	VolumesAttached   []VolumeAttached     `json:"os-extended-volumes:volumes_attached,omitempty"`
	Locked            bool                 `json:"locked,omitempty"`
	Image             string               `json:"image,omitempty"`
	AccessIPv4        string               `json:"accessIPv4,omitempty"`
	AccessIPv6        string               `json:"accessIPv6,omitempty"`
	Created           string               `json:"created,omitempty"`
	HostID            string               `json:"hostId,omitempty"`
	Tags              []string             `json:"tags,omitempty"`
	Flavor            Flavor               `json:"flavor,omitempty"`
	KeyName           string               `json:"key_name,omitempty"`
	VMState           string               `json:"OS-EXT-STS:vm_state,omitempty"`
	UserID            string               `json:"user_id,omitempty"`
	Name              string               `json:"name,omitempty"`
	Progress          int                  `json:"progress,omitempty"`
	LaunchedAt        string               `json:"OS-SRV-USG:launched_at,omitempty"`
	Updated           string               `json:"updated,omitempty"`
	Status            string               `json:"status,omitempty"`
	TerminatedAt      *string              `json:"OS-SRV-USG:terminated_at,omitempty"`
	ConfigDrive       string               `json:"config_drive,omitempty"`

	//added by control-plane node to allow worker nodes to check for respective CP
	ControlPlaneRef ControlPlaneRef `json:"controlPlaneRef,omitempty"`

	//this is for worker nodes to make sure it joined its respective cluster
	WorkerRef WorkerRef `json:"workerRef,omitempty"`
}
type WorkerRef struct {
	JoinedControlPlane bool `json:"joinedControlPlane,omitempty"`
}

type ControlPlaneRef struct {
	Type                string `json:"type,omitempty"`
	LastTransactionTime string `json:"lastTransactionTime,omitempty"`
	Status              bool   `json:"status,omitempty"`
}

// Supporting structs
type Links struct {
	Rel  string `json:"rel,omitempty"`
	Href string `json:"href,omitempty"`
}

type SecurityGroups struct {
	Name string `json:"name,omitempty"`
}

type Address struct {
	MACAddr string `json:"OS-EXT-IPS-MAC:mac_addr,omitempty"`
	Type    string `json:"OS-EXT-IPS:type,omitempty"`
	Addr    string `json:"addr,omitempty"`
	Version int    `json:"version,omitempty"`
}

type VolumeAttached struct {
	DeleteOnTermination bool   `json:"delete_on_termination,omitempty"`
	ID                  string `json:"id,omitempty"`
}

type Flavor struct {
	Disk       int               `json:"disk,omitempty"`
	Swap       int               `json:"swap,omitempty"`
	Original   string            `json:"original_name,omitempty"`
	ExtraSpecs map[string]string `json:"extra_specs,omitempty"`
	Ephemeral  int               `json:"ephemeral,omitempty"`
	VCPUs      int               `json:"vcpus,omitempty"`
	RAM        int               `json:"ram,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KTMachine is the Schema for the ktmachines API.
type KTMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KTMachineSpec   `json:"spec,omitempty"`
	Status KTMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KTMachineList contains a list of KTMachine.
type KTMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KTMachine `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &KTMachine{}, &KTMachineList{})
}
