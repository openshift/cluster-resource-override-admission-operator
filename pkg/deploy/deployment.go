package deploy

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	resourceensurer "github.com/openshift/cluster-resource-override-admission-operator/pkg/ensurer"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
)

func NewDeploymentInstall(lister listersappsv1.DeploymentLister, oc operatorruntime.OperandContext, asset *asset.Asset, ensurer *resourceensurer.DeploymentEnsurer) Interface {
	return &deployment{
		lister:  lister,
		context: oc,
		asset:   asset,
		ensurer: ensurer,
	}
}

type deployment struct {
	lister  listersappsv1.DeploymentLister
	context operatorruntime.OperandContext
	asset   *asset.Asset
	ensurer *resourceensurer.DeploymentEnsurer
}

func (d *deployment) Name() string {
	return d.asset.Deployment().Name()
}

func (d *deployment) IsAvailable() (available bool, err error) {
	name := d.asset.Deployment().Name()
	current, err := d.lister.Deployments(d.context.WebhookNamespace()).Get(name)
	if err != nil {
		return
	}

	available, err = GetDeploymentStatus(current)
	return
}

func (d *deployment) Get() (object runtime.Object, accessor metav1.Object, err error) {
	name := d.asset.Deployment().Name()
	object, err = d.lister.Deployments(d.context.WebhookNamespace()).Get(name)
	if err != nil {
		return
	}

	accessor, err = meta.Accessor(object)
	return
}

func (d *deployment) Ensure(parent, child Applier) (current runtime.Object, accessor metav1.Object, err error) {
	desired := d.asset.Deployment().New()

	if parent != nil {
		parent.Apply(desired)
	}
	if child != nil {
		child.Apply(&desired.Spec.Template)
	}

	current, err = d.ensurer.Ensure(desired)
	if err != nil {
		return
	}

	accessor, err = meta.Accessor(current)
	return
}
