package runtime

import (
	"fmt"

	dynamicclient "github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	apiregistrationclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

type Client struct {
	Operator        versioned.Interface
	Kubernetes      kubernetes.Interface
	APIRegistration apiregistrationclientset.Interface
	APIExtension    apiextensionsclientset.Interface
	Dynamic         dynamicclient.Ensurer
	RawDynamic      dynamic.Interface
}

func NewClient(config *rest.Config) (clients *Client, err error) {
	operator, buildErr := versioned.NewForConfig(config)
	if buildErr != nil {
		err = fmt.Errorf("failed to construct client for autoscaling.openshift.io - %s", buildErr.Error())
		return
	}

	kubeclient, buildErr := kubernetes.NewForConfig(config)
	if buildErr != nil {
		err = fmt.Errorf("failed to construct client for kubernetes - %s", buildErr.Error())
		return
	}

	rawDynamic, buildErr := dynamic.NewForConfig(config)
	if buildErr != nil {
		err = fmt.Errorf("failed to construct dynamic client - %s", buildErr.Error())
		return
	}

	apiregistration, buildErr := apiregistrationclientset.NewForConfig(config)
	if buildErr != nil {
		err = fmt.Errorf("failed to construct apiregistration client - %s", buildErr.Error())
		return
	}

	apiextension, buildErr := apiextensionsclientset.NewForConfig(config)
	if buildErr != nil {
		err = fmt.Errorf("failed to construct apiextension client - %s", buildErr.Error())
		return
	}

	clients = &Client{
		Operator:        operator,
		Kubernetes:      kubeclient,
		APIRegistration: apiregistration,
		APIExtension:    apiextension,
		Dynamic:         dynamicclient.NewEnsurer(rawDynamic),
		RawDynamic:      rawDynamic,
	}

	return
}
