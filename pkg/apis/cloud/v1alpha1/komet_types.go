package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KometSpec defines the desired state of Komet
// +k8s:openapi-gen=true
type KometSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	GitSource string `json:"gitSource"` 
	GitBranch string `json:"gitBranch"`
	DockerImage string `json:"dockerImage"`
	Version string `json:"version"`
	Author string `json:"author"`
}

// KometStatus defines the observed state of Komet
// +k8s:openapi-gen=true
type KometStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Komet is the Schema for the komets API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Komet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KometSpec   `json:"spec,omitempty"`
	Status KometStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KometList contains a list of Komet
type KometList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Komet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Komet{}, &KometList{})
}
