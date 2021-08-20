package secondarywatch

import (
	admissionregistrationv1 "k8s.io/client-go/listers/admissionregistration/v1"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
)

// Lister is a set of Lister(s) for secondary resource(s)
type Lister struct {
	deployment     listersappsv1.DeploymentLister
	daemonset      listersappsv1.DaemonSetLister
	pod            listerscorev1.PodLister
	configmap      listerscorev1.ConfigMapLister
	service        listerscorev1.ServiceLister
	secret         listerscorev1.SecretLister
	serviceaccount listerscorev1.ServiceAccountLister
	webhook        admissionregistrationv1.MutatingWebhookConfigurationLister
}

func (l *Lister) CoreV1ConfigMapLister() listerscorev1.ConfigMapLister {
	return l.configmap
}

func (l *Lister) CoreV1SecretLister() listerscorev1.SecretLister {
	return l.secret
}

func (l *Lister) CoreV1ServiceLister() listerscorev1.ServiceLister {
	return l.service
}

func (l *Lister) AppsV1DeploymentLister() listersappsv1.DeploymentLister {
	return l.deployment
}

func (l *Lister) AppsV1DaemonSetLister() listersappsv1.DaemonSetLister {
	return l.daemonset
}

func (l *Lister) AdmissionRegistrationV1MutatingWebhookConfigurationLister() admissionregistrationv1.MutatingWebhookConfigurationLister {
	return l.webhook
}
