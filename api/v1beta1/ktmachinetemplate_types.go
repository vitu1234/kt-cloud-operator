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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KTMachineTemplateSpec defines the desired state of KTMachineTemplate.
type KTMachineTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KTMachineTemplate. Edit ktmachinetemplate_types.go to remove/update
	Template Template `json:"template,omitempty"`
}

// define the properties of a machine
// Template defines the properties of a machine
type Template struct {
	Spec Spec `json:"spec,omitempty"`
}

// Spec holds details about the machine specification
type Spec struct {
	Flavor             string               `json:"flavor,omitempty"`
	SSHKeyName         string               `json:"sshKeyName,omitempty"`
	BlockDeviceMapping []BlockDeviceMapping `json:"blockDeviceMapping,omitempty"`
	NetworkTier        []NetworkTier        `json:"networkTier,omitempty"`
	Ports              []Port               `json:"ports,omitempty"`
}

type BlockDeviceMapping struct {
	DestinationType string `json:"destinationType,omitempty"`
	BootIndex       int    `json:"bootIndex,omitempty"`
	SourceType      string `json:"sourceType,omitempty"`
	VolumeSize      int    `json:"volumeSize,omitempty"`
	ID              string `json:"id,omitempty"`
}

type NetworkTier struct {
	ID string `json:"id,omitempty"`
}

// Port defines a network configuration or IP details
type Port struct {
	Network  *Network  `json:"network,omitempty"`
	FixedIPs []FixedIP `json:"fixedIPs,omitempty"`
}

// Network holds the network details such as name, tags, or ID
type Network struct {
	Name string `json:"name,omitempty"`
	Tags string `json:"tags,omitempty"`
	ID   string `json:"id,omitempty"`
}

// FixedIP represents fixed IP information with subnet details
type FixedIP struct {
	Subnet *Subnet `json:"subnet,omitempty"`
}

// Subnet holds the subnet ID information
type Subnet struct {
	ID string `json:"id,omitempty"`
}

// KTMachineTemplateStatus defines the observed state of KTMachineTemplate.
type KTMachineTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KTMachineTemplate is the Schema for the ktmachinetemplates API.
type KTMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KTMachineTemplateSpec   `json:"spec,omitempty"`
	Status KTMachineTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KTMachineTemplateList contains a list of KTMachineTemplate.
type KTMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KTMachineTemplate `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &KTMachineTemplate{}, &KTMachineTemplateList{})
}
