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

const (
	// KTSubjectTokenFinalizer allows ReconcileKTMachine to clean up resources associated with KTVM before
	// removing it from the apiserver.
	KTSubjectTokenFinalizer = "ktsubjecttoken.infrastructure.dcnlab.ssu.ac.kr"
)

// KTSubjectTokenSpec defines the desired state of KTSubjectToken.
type KTSubjectTokenSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KTSubjectToken. Edit ktsubjecttoken_types.go to remove/update
	SubjectToken string `json:"subjectToken,omitempty"`
	Token        Token  `json:"token,omitempty"`
	// Date         string `json:"date,omitempty"`
	ClusterRef ClusterRef `json:"clusterRef,omitempty"`
}

type ClusterRef struct {
	ApiVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}

type Token struct {
	ExpiresAt string `json:"expiresAt,omitempty"`
	IsDomain  bool   `json:"isDomain,omitempty"`
}

// KTSubjectTokenStatus defines the observed state of KTSubjectToken.
type KTSubjectTokenStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	SubjectToken string `json:"subjectToken,omitempty"`
	Token        Token  `json:"token,omitempty"`
	CreatedAt    string `json:"createdAt,omitempty"` //The time logged successfully from KTCloud
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KTSubjectToken is the Schema for the ktsubjecttokens API.
type KTSubjectToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KTSubjectTokenSpec   `json:"spec,omitempty"`
	Status KTSubjectTokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KTSubjectTokenList contains a list of KTSubjectToken.
type KTSubjectTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KTSubjectToken `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &KTSubjectToken{}, &KTSubjectTokenList{})
	// utilruntime.Must(AddToScheme(runtime.NewScheme())) // Ensure it's registered

}
