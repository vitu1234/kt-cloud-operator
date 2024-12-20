/*
Copyright 2024.

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

// KTMachineSpec defines the desired state of KTMachine.
type KTMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KTMachine. Edit ktmachine_types.go to remove/update
	Flavor             string               `json:"flavor,omitempty"`
	SSHKeyName         string               `json:"sshKeyName,omitempty"`
	BlockDeviceMapping []BlockDeviceMapping `json:"blockDeviceMapping,omitempty"`
	NetworkTier        []NetworkTier        `json:"networkTier,omitempty"`
	Networks           []Networks           `json:"networks,omitempty"`
	Ports              []Port               `json:"ports,omitempty"`
	AvailabilityZone   string               `json:"availabilityZone,omitempty"`
	UserData           string               `json:"userData,omitempty"`
}

type Networks struct {
	ID string `json:"id,omitempty"`
}

// KTMachineStatus defines the observed state of KTMachine.
type KTMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ID             string           `json:"id,omitempty"`
	AdminPass      string           `json:"adminPass,omitempty"`
	State          string           `json:"state,omitempty"`
	Links          []Links          `json:"links,omitempty"`
	SecurityGroups []SecurityGroups `json:"securityGroups,omitempty"`
}

type Links struct {
	Rel  string `json:"rel,omitempty"`
	Href string `json:"href,omitempty"`
}

type SecurityGroups struct {
	Name string `json:"name,omitempty"`
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
