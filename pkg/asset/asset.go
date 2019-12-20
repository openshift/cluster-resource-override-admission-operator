package asset

import (
	"fmt"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
)

func New(context runtime.OperandContext) *Asset {
	values := &Values{
		Name:                           context.WebhookName(),
		Namespace:                      context.WebhookNamespace(),
		ServiceAccountName:             context.WebhookName(),
		OperandImage:                   context.OperandImage(),
		OperandVersion:                 context.OperandVersion(),
		AdmissionAPIGroup:              "admission.autoscaling.openshift.io",
		AdmissionAPIVersion:            "v1",
		AdmissionAPIResource:           "clusterresourceoverrides",
		OwnerLabelKey:                  "operator.autoscaling.openshift.io/clusterresourceoverride",
		OwnerLabelValue:                "true",
		SelectorLabelKey:               "clusterresourceoverride",
		SelectorLabelValue:             "true",
		ConfigurationKey:               "configuration.yaml",
		ConfigurationHashAnnotationKey: fmt.Sprintf("%s.%s/configuration.hash", context.WebhookName(), autoscaling.GroupName),
		ServingCertHashAnnotationKey:   fmt.Sprintf("%s.%s/servingcert.hash", context.WebhookName(), autoscaling.GroupName),
		OwnerAnnotationKey:             fmt.Sprintf("%s.%s/owner", context.WebhookName(), autoscaling.GroupName),
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
