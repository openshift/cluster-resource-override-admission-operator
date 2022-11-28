package asset

import (
	"fmt"

	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
)

const (
	WebhookName      = "podsvtoverride.admission.node.openshift.io"
	APIGroup         = "admission.node.openshift.io"
	APIGroupVersion  = "v1"
	APIGroupResource = "podsvtoverride"
)

func New(context runtime.OperandContext) *Asset {
	values := &Values{
		Name:                           WebhookName,
		Namespace:                      context.WebhookNamespace(),
		ServiceAccountName:             WebhookName,
		OperandImage:                   context.OperandImage(),
		OperandVersion:                 context.OperandVersion(),
		AdmissionAPIGroup:              APIGroup,
		AdmissionAPIVersion:            APIGroupVersion,
		AdmissionAPIResource:           APIGroupResource,
		OwnerLabelKey:                  "operator.autoscaling.openshift.io/clusterresourceoverride",
		OwnerLabelValue:                "true",
		SelectorLabelKey:               "podsvtoverride",
		SelectorLabelValue:             "true",
		ConfigurationKey:               "configuration.yaml",
		ConfigurationHashAnnotationKey: fmt.Sprintf("%s.%s/configuration.hash", WebhookName, selinuxfixv1.GroupName),
		ServingCertHashAnnotationKey:   fmt.Sprintf("%s.%s/servingcert.hash", WebhookName, selinuxfixv1.GroupName),
		OwnerAnnotationKey:             fmt.Sprintf("%s.%s/owner", WebhookName, selinuxfixv1.GroupName),
	}

	return &Asset{
		values: values,
	}
}

type Asset struct {
	values *Values
}

func (a *Asset) Values() *Values {
	return a.values
}

type Values struct {
	Name                 string
	Namespace            string
	ServiceAccountName   string
	OperandImage         string
	OperandVersion       string
	AdmissionAPIGroup    string
	AdmissionAPIVersion  string
	AdmissionAPIResource string
	OwnerLabelKey        string
	OwnerLabelValue      string
	SelectorLabelKey     string
	SelectorLabelValue   string
	ConfigurationKey     string

	ConfigurationHashAnnotationKey string
	ServingCertHashAnnotationKey   string
	OwnerAnnotationKey             string
}
