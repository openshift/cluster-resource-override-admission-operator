package deploy

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/ensurer"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
)

func NewDaemonSetInstall(lister listersappsv1.DaemonSetLister, oc operatorruntime.OperandContext, asset *asset.Asset, deployment *ensurer.DaemonSetEnsurer) Interface {
	return &daemonset{
		lister:     lister,
		context:    oc,
		asset:      asset,
		deployment: deployment,
	}
}

type daemonset struct {
	lister     listersappsv1.DaemonSetLister
	context    operatorruntime.OperandContext
	asset      *asset.Asset
	deployment *ensurer.DaemonSetEnsurer
}

func (d *daemonset) Name() string {
	return d.asset.DaemonSet().Name()
}

func (d *daemonset) IsAvailable() (available bool, err error) {
	name := d.asset.DaemonSet().Name()
	current, err := d.lister.DaemonSets(d.context.WebhookNamespace()).Get(name)
	if err != nil {
		return
	}

	available, err = GetDaemonSetStatus(current)
	return
}

func (d *daemonset) Get() (object runtime.Object, accessor metav1.Object, err error) {
	name := d.asset.DaemonSet().Name()
	object, err = d.lister.DaemonSets(d.context.WebhookNamespace()).Get(name)
	if err != nil {
		return
	}

	accessor, err = meta.Accessor(object)
	return
}

func (d *daemonset) Ensure(parent, child Applier) (current runtime.Object, accessor metav1.Object, err error) {
	desired := d.asset.DaemonSet().New()

	if parent != nil {
		parent.Apply(desired)
	}
	if child != nil {
		child.Apply(&desired.Spec.Template)
	}

	current, err = d.deployment.Ensure(desired)
	if err != nil {
		return
	}

	accessor, err = meta.Accessor(current)
	return
}
