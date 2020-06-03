package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SeleniumGridSpec defines the desired state of SeleniumGrid
type SeleniumGridSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	ChromeNodes int32  `json:"chromeNodes"`
	HubVersion  string `json:"hubversion"`
	HubMemory   string `json:"hubmemory"`
	HubCPU      string `json:"hubcpu"`
}

// SeleniumGridStatus defines the observed state of SeleniumGrid
type SeleniumGridStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	ChromeNodeList []string `json:"nodes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SeleniumGrid is the Schema for the seleniumgrids API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=seleniumgrids,scope=Namespaced
type SeleniumGrid struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SeleniumGridSpec   `json:"spec,omitempty"`
	Status SeleniumGridStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SeleniumGridList contains a list of SeleniumGrid
type SeleniumGridList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SeleniumGrid `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SeleniumGrid{}, &SeleniumGridList{})
}
