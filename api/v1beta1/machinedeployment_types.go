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

// MachineDeploymentSpec defines the desired state of MachineDeployment.
type MachineDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MachineDeployment. Edit machinedeployment_types.go to remove/update
	// ClusterName string      `json:"clusterName"`
	Replicas int         `json:"replicas"`
	Type     string      `json:"type"` //control-plane or worker
	Selector Selector    `json:"selector,omitempty"`
	Template MachineSpec `json:"template,omitempty"`
}

// Selector defines the labels used for matching machines
type Selector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

type MachineSpec struct {
	Spec MachineSpecDetails `json:"spec,omitempty"`
}

// MachineSpecDetails holds detailed specifications for a machine
type MachineSpecDetails struct {
	Bootstrap         Bootstrap         `json:"bootstrap,omitempty"`
	ClusterName       string            `json:"clusterName,omitempty"`
	FailureDomain     string            `json:"failureDomain,omitempty"`
	InfrastructureRef InfrastructureRef `json:"infrastructureRef,omitempty"`
	Version           string            `json:"version,omitempty"`
}

// Bootstrap holds bootstrap configuration reference
type Bootstrap struct {
	ConfigRef ConfigRef `json:"configRef,omitempty"`
}

// ConfigRef defines the reference to a bootstrap configuration
type ConfigRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}

// InfrastructureRef defines the infrastructure reference details
type InfrastructureRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}

// MachineDeploymentStatus defines the observed state of MachineDeployment.
type MachineDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MachineDeployment is the Schema for the machinedeployments API.
type MachineDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineDeploymentSpec   `json:"spec,omitempty"`
	Status MachineDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MachineDeploymentList contains a list of MachineDeployment.
type MachineDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineDeployment `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &MachineDeployment{}, &MachineDeploymentList{})
}
