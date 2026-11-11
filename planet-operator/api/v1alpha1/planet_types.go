package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PlanetSpec décrit une planète et, éventuellement, l'image qui la sert.
type PlanetSpec struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// +kubebuilder:validation:Minimum=0
	MassKg float64 `json:"massKg"`

	// +kubebuilder:validation:Minimum=0
	RadiusKm float64 `json:"radiusKm"`

	// +kubebuilder:validation:Minimum=0
	OrbitDays int32 `json:"orbitDays"`

	// Message affiché par la page de la planète.
	// +optional
	Message string `json:"message,omitempty"`

	// Image du conteneur de la planète. Si vide, une page nginx générée est utilisée.
	// Format attendu : registre/nom:tag, ou registre/nom@sha256:<64 hex>.
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-z0-9.\-]+(/[a-z0-9._\-]+)+(:[A-Za-z0-9._\-]+)?(@sha256:[a-f0-9]{64})?$`
	Image string `json:"image,omitempty"`
}

// PlanetStatus reflète l'état observé de la planète.
type PlanetStatus struct {
	// +optional
	Phase string `json:"phase,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Planet est la ressource personnalisée décrivant une planète.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
type Planet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlanetSpec   `json:"spec,omitempty"`
	Status PlanetStatus `json:"status,omitempty"`
}

// PlanetList contient une liste de Planet.
// +kubebuilder:object:root=true
type PlanetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Planet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Planet{}, &PlanetList{})
}
