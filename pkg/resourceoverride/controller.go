package resourceoverride

import (
	"context"
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/controller"
	listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/resourceoverride/internal/reconciler"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
)

const (
	ControllerName = "resourceoverride"
)

type Options struct {
	ResyncPeriod time.Duration
	Workers      int
	Client       *operatorruntime.Client
}

func New(options *Options) (c controller.Interface, err error) {
	if options == nil || options.Client == nil {
		err = errors.New("Invalid input to controller.New")
		return
	}

	client := options.Client.Operator
	watcher := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return client.AutoscalingV1().ResourceOverrides("").List(context.TODO(), options)
		},

		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.AutoscalingV1().ResourceOverrides("").Watch(context.TODO(), options)
		},
	}

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	store, informer := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: watcher,
		ObjectType:    &autoscalingv1.ResourceOverride{},
		Handler:       controller.NewEventHandler(queue),
		ResyncPeriod:  options.ResyncPeriod,
		Indexers:      cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	})

	lister := listers.NewResourceOverrideLister(store.(cache.Indexer))

	reconciler := reconciler.NewReconciler(client, lister)

	c = &resourceOverrideController{
		workers:    options.Workers,
		queue:      queue,
		informer:   informer,
		reconciler: reconciler,
		lister:     lister,
	}

	return
}

type resourceOverrideController struct {
	workers    int
	queue      workqueue.RateLimitingInterface
	informer   cache.Controller
	reconciler controllerreconciler.Reconciler
	lister     listers.ResourceOverrideLister
}

func (c *resourceOverrideController) Name() string {
	return ControllerName
}

func (c *resourceOverrideController) WorkerCount() int {
	return c.workers
}

func (c *resourceOverrideController) Queue() workqueue.RateLimitingInterface {
	return c.queue
}

func (c *resourceOverrideController) Informer() cache.Controller {
	return c.informer
}

func (c *resourceOverrideController) Reconciler() controllerreconciler.Reconciler {
	return c.reconciler
}
