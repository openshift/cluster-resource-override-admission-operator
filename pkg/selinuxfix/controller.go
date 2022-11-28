package selinuxfix

import (
	"context"
	"errors"
	"time"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/deploy"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/ensurer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"

	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset-selinux"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/controller"
	listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/selinuxfix/v1"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/selinuxfix/internal/handlers"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/selinuxfix/internal/reconciler"
)

const (
	ControllerName = "podsvtoverride"
)

type Options struct {
	ResyncPeriod   time.Duration
	Workers        int
	Client         *operatorruntime.Client
	RuntimeContext operatorruntime.OperandContext
	Lister         *secondarywatch.Lister
}

func New(options *Options) (c controller.Interface, e operatorruntime.Enqueuer, err error) {
	if options == nil || options.Client == nil || options.RuntimeContext == nil {
		err = errors.New("invalid input to controller.New")
		return
	}

	// Create a new ClusterResourceOverrides watcher
	client := options.Client.Operator
	watcher := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return client.SelinuxfixV1().SelinuxFixOverrides().List(context.TODO(), options)
		},

		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.SelinuxfixV1().SelinuxFixOverrides().Watch(context.TODO(), options)
		},
	}

	// We need a queue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Bind the work queue to a cache with the help of an informer. This way we
	// make sure that whenever the cache is updated, the ClusterResourceOverride
	// key is added to the work queue.
	// Note that when we finally process the item from the workqueue, we might
	// see a newer version of the ClusterResourceOverride than the version which
	// was responsible for triggering the update.
	indexer, informer := cache.NewIndexerInformer(watcher, &selinuxfixv1.SelinuxFixOverride{}, options.ResyncPeriod,
		controller.NewEventHandler(queue), cache.Indexers{})

	lister := listers.NewSelinuxFixOverrideLister(indexer)

	// setup operand asset
	operandAsset := asset.New(options.RuntimeContext)

	// initialize install strategy, we use daemonset
	d := deploy.NewDaemonSetInstallForSelinux(options.Lister.AppsV1DaemonSetLister(), options.RuntimeContext, operandAsset, ensurer.NewDaemonSetEnsurer(options.Client.Dynamic))

	reconciler := reconciler.NewReconciler(&handlers.Options{
		OperandContext:  options.RuntimeContext,
		Client:          options.Client,
		PrimaryLister:   lister,
		SecondaryLister: options.Lister,
		Asset:           operandAsset,
		Deploy:          d,
	})

	c = &selinuxFixOverrideController{
		workers:    options.Workers,
		queue:      queue,
		informer:   informer,
		reconciler: reconciler,
		lister:     lister,
	}
	e = &enqueuer{
		queue:              queue,
		lister:             lister,
		ownerAnnotationKey: operandAsset.Values().OwnerAnnotationKey,
	}

	return
}

type selinuxFixOverrideController struct {
	workers    int
	queue      workqueue.RateLimitingInterface
	informer   cache.Controller
	reconciler controllerreconciler.Reconciler
	lister     listers.SelinuxFixOverrideLister
}

func (c *selinuxFixOverrideController) Name() string {
	return ControllerName
}

func (c *selinuxFixOverrideController) WorkerCount() int {
	return c.workers
}

func (c *selinuxFixOverrideController) Queue() workqueue.RateLimitingInterface {
	return c.queue
}

func (c *selinuxFixOverrideController) Informer() cache.Controller {
	return c.informer
}

func (c *selinuxFixOverrideController) Reconciler() controllerreconciler.Reconciler {
	return c.reconciler
}
