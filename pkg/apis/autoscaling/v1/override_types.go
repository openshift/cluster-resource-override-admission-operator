package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceOverrideKind = "ResourceOverride"
)

type ResourceOverrideConditionType string

const (
	ValidationFailure ResourceOverrideConditionType = "ValidationFailure"
)

const (
	InvalidParameters = "InvalidParameters"
	ExemptNamespace   = "ExemptNamespace"
)

type ResourceOverrideCondition struct {
	// Type is the type of ResourceOverride condition.
	Type ResourceOverrideConditionType `json:"type" description:"type of ResourceOverride condition"`

	// Status is the status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// Reason is a one-word CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// Message is a human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`

	// LastTransitionTime is the last time the condition transit from one status to another
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" description:"last time the condition transit from one status to another" hash:"ignore"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ro

type ResourceOverride struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceOverrideSpec   `json:"spec,omitempty"`
	Status ResourceOverrideStatus `json:"status,omitempty"`
}

type ResourceOverrideSpec struct {
	PodResourceOverride PodResourceOverrideSpec `json:"podResourceOverride"`
	// +optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
}

type ResourceOverrideStatus struct {
	Conditions []ResourceOverrideCondition `json:"conditions,omitempty" hash:"set"`
}

// PodResourceOverrideSpec is the configuration for the ResourceOverride
// admission controller which overrides user-provided container request/limit values.
type PodResourceOverrideSpec struct {
	// For each of the following, if a non-zero ratio is specified then the initial
	// value (if any) in the pod spec is overwritten according to the ratio.
	// LimitRange defaults are merged prior to the override.
	//

	// ForceSelinuxRelabel (if true) label pods with spc_t if they have a PVC
	// +optional
	ForceSelinuxRelabel bool `json:"forceSelinuxRelabel,omitempty"`

	// LimitCPUToMemoryPercent (if > 0) overrides the CPU limit to a ratio of the memory limit;
	// 100% overrides CPU to 1 core per 1GiB of RAM. This is done before overriding the CPU request.
	// +kubebuilder:validation:Minimum=0
	LimitCPUToMemoryPercent int64 `json:"limitCPUToMemoryPercent,omitempty"`

	// CPURequestToLimitPercent (if > 0) overrides CPU request to a percentage of CPU limit
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	CPURequestToLimitPercent int64 `json:"cpuRequestToLimitPercent,omitempty"`

	// MemoryRequestToLimitPercent (if > 0) overrides memory request to a percentage of memory limit
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	MemoryRequestToLimitPercent int64 `json:"memoryRequestToLimitPercent,omitempty"`

	// CPURequestToRequestPercent (if > 0) overrides CPU request to a percentage of the
	// existing CPU request.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	CPURequestToRequestPercent int64 `json:"cpuRequestToRequestPercent,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ResourceOverrideList contains a list of ResourceOverrides.
type ResourceOverrideList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceOverride `json:"items"`
}
