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

// KubeadmConfigTemplateSpec defines the desired state of KubeadmConfigTemplate.
type KubeadmConfigTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KubeadmConfigTemplate. Edit kubeadmconfigtemplate_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// KubeadmConfigTemplateStatus defines the observed state of KubeadmConfigTemplate.
type KubeadmConfigTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KubeadmConfigTemplate is the Schema for the kubeadmconfigtemplates API.
type KubeadmConfigTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubeadmConfigTemplateSpec   `json:"spec,omitempty"`
	Status KubeadmConfigTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KubeadmConfigTemplateList contains a list of KubeadmConfigTemplate.
type KubeadmConfigTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeadmConfigTemplate `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &KubeadmConfigTemplate{}, &KubeadmConfigTemplateList{})
}
