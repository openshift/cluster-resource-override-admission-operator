package helper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	webhookName = "clusterresourceoverrides.admission.autoscaling.openshift.io"
)

type PreCondition struct {
	Client kubernetes.Interface
}

func (f *PreCondition) MustHaveAdmissionRegistrationV1(t *testing.T) {
	apiGroupList := &metav1.APIGroupList{}
	err := f.Client.Discovery().RESTClient().Get().AbsPath("/apis").Do(context.TODO()).Into(apiGroupList)
	require.NoError(t, err, "fetching /apis")

	t.Log("finding the admissionregistration.k8s.io API group in the /apis discovery document")

	var group *metav1.APIGroup
	for _, g := range apiGroupList.Groups {
		if g.Name == admissionregistrationv1.GroupName {
			group = &g
			break
		}
	}

	require.NotNil(t, group, "admissionregistration.k8s.io API group not found in /apis discovery document")

	t.Log("finding the admissionregistration.k8s.io/v1 API group/version in the /apis discovery document")
	var version *metav1.GroupVersionForDiscovery
	for _, v := range group.Versions {
		if v.Version == admissionregistrationv1.SchemeGroupVersion.Version {
			version = &v
			break
		}
	}

	require.NotNil(t, version, "admissionregistration.k8s.io/v1 API group version not found in /apis discovery document")
}

func (f *PreCondition) MustHaveClusterResourceOverrideAdmissionConfiguration(t *testing.T) {
	t.Logf("fetching MutatingWebhookConfigurations %s", webhookName)

	configuration, err := f.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), webhookName, metav1.GetOptions{})

	require.NoErrorf(t, err, "MutatingWebhookConfiguration %s resource not found in /apis/admissionregistration.k8s.io/v1 discovery document", webhookName)
	require.NotNil(t, configuration)
}
