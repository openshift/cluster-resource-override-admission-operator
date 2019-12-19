package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterResourceOverrideKind = "ClusterResourceOverride"
)

type ClusterResourceOverrideConditionType string

const (
	InstallReadinessFailure ClusterResourceOverrideConditionType = "InstallReadinessFailure"
	Available               ClusterResourceOverrideConditionType = "Available"
)

const (
	InvalidParameters            = "InvalidParameters"
	ConfigurationCheckFailed     = "ConfigurationCheckFailed"
	CertNotAvailable             = "CertNotAvailable"
	CannotSetReference           = "CannotSetReference"
	CannotGenerateCert           = "CannotGenerateCert"
	InternalError                = "InternalError"
	AdmissionWebhookNotAvailable = "AdmissionWebhookNotAvailable"
	DeploymentNotReady           = "DeploymentNotReady"
)

type ClusterResourceOverrideCondition struct {
	// Type is the type of ClusterResourceOverride condition.
	Type ClusterResourceOverrideConditionType `json:"type" description:"type of ClusterResourceOverride condition"`

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
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterResourceOverride struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterResourceOverrideSpec   `json:"spec,omitempty"`
	Status ClusterResourceOverrideStatus `json:"status,omitempty"`
}

type ClusterResourceOverrideSpec struct {
	PodResourceOverride PodResourceOverride `json:"podResourceOverride"`
}

type ClusterResourceOverrideStatus struct {
	// Resources is a set of resources associated with the operand.
	Resources  ClusterResourceOverrideResources    `json:"resources,omitempty"`
	Hash       ClusterResourceOverrideResourceHash `json:"hash,omitempty"`
	Conditions []ClusterResourceOverrideCondition  `json:"conditions,omitempty" hash:"set"`
	Version    string                              `json:"version,omitempty"`
	Image      string                              `json:"image,omitempty"`

	// CertsRotateAt is the time the serving certs will be rotated at.
	// +optional
	CertsRotateAt metav1.Time `json:"certsRotateAt,omitempty"`
}

type ClusterResourceOverrideResourceHash struct {
	Configuration string `json:"configuration,omitempty"`
	ServingCert   string `json:"servingCert,omitempty"`
}

type ClusterResourceOverrideResources struct {
	// ConfigurationRef points to the ConfigMap that contains the parameters for
	// ClusterResourceOverride admission webhook.
	ConfigurationRef *corev1.ObjectReference `json:"configurationRef,omitempty"`

	// ServiceCAConfigMapRef points to the ConfigMap that is injected with a
	// data item (key "service-ca.crt") containing the PEM-encoded CA signing bundle.
	ServiceCAConfigMapRef *corev1.ObjectReference `json:"serviceCAConfigMapRef,omitempty"`

	// ServiceRef points to the Service object that exposes the ClusterResourceOverride
	// webhook admission server to the cluster.
	// This service is annotated with `service.beta.openshift.io/serving-cert-secret-name`
	// so that service-ca operator can issue a signed serving certificate/key pair.
	ServiceRef *corev1.ObjectReference `json:"serviceRef,omitempty"`

	// ServiceCertSecretRef points to the Secret object which is created by the
	// service-ca operator and contains the signed serving certificate/key pair.
	ServiceCertSecretRef *corev1.ObjectReference `json:"serviceCertSecretRef,omitempty"`

	// DeploymentRef points to the Deployment object of the ClusterResourceOverride
	// admission webhook server.
	DeploymentRef *corev1.ObjectReference `json:"deploymentRef,omitempty"`

	// APiServiceRef points to the APIService object related to the ClusterResourceOverride
	// admission webhook server.
	APiServiceRef *corev1.ObjectReference `json:"apiServiceRef,omitempty"`

	// APiServiceRef points to the APIService object related to the ClusterResourceOverride
	// admission webhook server.
	MutatingWebhookConfigurationRef *corev1.ObjectReference `json:"mutatingWebhookConfigurationRef,omitempty"`
}

// PodResourceOverride is the configuration for the admission controller which
// overrides user-provided container request/limit values.
type PodResourceOverride struct {
	metav1.TypeMeta `json:",inline"`
	Spec            PodResourceOverrideSpec `json:"spec,omitempty"`
}

// PodResourceOverrideSpec is the configuration for the ClusterResourceOverride
// admission controller which overrides user-provided container request/limit values.
type PodResourceOverrideSpec struct {
	// For each of the following, if a non-zero ratio is specified then the initial
	// value (if any) in the pod spec is overwritten according to the ratio.
	// LimitRange defaults are merged prior to the override.
	//

	// LimitCPUToMemoryPercent (if > 0) overrides the CPU limit to a ratio of the memory limit;
	// 100% overrides CPU to 1 core per 1GiB of RAM. This is done before overriding the CPU request.
	LimitCPUToMemoryPercent int64 `json:"limitCPUToMemoryPercent"`

	// CPURequestToLimitPercent (if > 0) overrides CPU request to a percentage of CPU limit
	CPURequestToLimitPercent int64 `json:"cpuRequestToLimitPercent"`

	// MemoryRequestToLimitPercent (if > 0) overrides memory request to a percentage of memory limit
	MemoryRequestToLimitPercent int64 `json:"memoryRequestToLimitPercent"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterResourceOverrideList contains a list of IngressControllers.
type ClusterResourceOverrideList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterResourceOverride `json:"items"`
}
