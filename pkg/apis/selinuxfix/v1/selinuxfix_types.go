package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SelinuxFixOVerrideKind = "SelinuxFixOverride"
)

type SelinuxFixOverrideConditionType string

const (
	InstallReadinessFailure SelinuxFixOverrideConditionType = "InstallReadinessFailure"
	Available               SelinuxFixOverrideConditionType = "Available"
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

type SelinuxFixOverrideCondition struct {
	// Type is the type of ClusterResourceOverride condition.
	Type SelinuxFixOverrideConditionType `json:"type" description:"type of SelinuxFixOverride condition"`

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

type SelinuxFixOverride struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SelinuxFixOverrideSpec   `json:"spec,omitempty"`
	Status SelinuxFixOverrideStatus `json:"status,omitempty"`
}

type SelinuxFixOverrideSpec struct {
	Enabled bool `json:"enabled"`
}

type SelinuxFixOverrideStatus struct {
	Hash       SelinuxFixOverrideResourceHash `json:"hash,omitempty"`
	Conditions []SelinuxFixOverrideCondition  `json:"conditions,omitempty" hash:"set"`
	Version    string                         `json:"version,omitempty"`
	Image      string                         `json:"image,omitempty"`

	Resources SelinuxFixOverrideResources `json:"resources,omitempty"`

	// CertsRotateAt is the time the serving certs will be rotated at.
	// +optional
	CertsRotateAt metav1.Time `json:"certsRotateAt,omitempty"`
}

type SelinuxFixOverrideResources struct {
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

type SelinuxFixOverrideResourceHash struct {
	ServingCert string `json:"servingCert,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SelinuxFixOverrideList contains a list of IngressControllers.
type SelinuxFixOverrideList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SelinuxFixOverride `json:"items"`
}
