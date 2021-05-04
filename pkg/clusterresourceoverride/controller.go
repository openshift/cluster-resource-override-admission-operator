package clusterresourceoverride

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

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/handlers"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/reconciler"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/controller"
	autoscalingv1listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
)

const (
	ControllerName = "clusterresourceoverride"
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
			return client.AutoscalingV1().ClusterResourceOverrides().List(context.TODO(), options)
		},

		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.AutoscalingV1().ClusterResourceOverrides().Watch(context.TODO(), options)
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
	indexer, informer := cache.NewIndexerInformer(watcher, &autoscalingv1.ClusterResourceOverride{}, options.ResyncPeriod,
		controller.NewEventHandler(queue), cache.Indexers{})

	lister := listers.NewClusterResourceOverrideLister(indexer)

	// setup operand asset
	operandAsset := asset.New(options.RuntimeContext)

	// initialize install strategy, we use daemonset
	d := deploy.NewDaemonSetInstall(options.Lister.AppsV1DaemonSetLister(), options.RuntimeContext, operandAsset, ensurer.NewDaemonSetEnsurer(options.Client.Dynamic))

	reconciler := reconciler.NewReconciler(&handlers.Options{
		OperandContext:  options.RuntimeContext,
		Client:          options.Client,
		PrimaryLister:   lister,
		SecondaryLister: options.Lister,
		Asset:           operandAsset,
		Deploy:          d,
	})

	c = &clusterResourceOverrideController{
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

type clusterResourceOverrideController struct {
	workers    int
	queue      workqueue.RateLimitingInterface
	informer   cache.Controller
	reconciler controllerreconciler.Reconciler
	lister     autoscalingv1listers.ClusterResourceOverrideLister
}

func (c *clusterResourceOverrideController) Name() string {
	return ControllerName
}

func (c *clusterResourceOverrideController) WorkerCount() int {
	return c.workers
}

func (c *clusterResourceOverrideController) Queue() workqueue.RateLimitingInterface {
	return c.queue
}

func (c *clusterResourceOverrideController) Informer() cache.Controller {
	return c.informer
}

func (c *clusterResourceOverrideController) Reconciler() controllerreconciler.Reconciler {
	return c.reconciler
}
