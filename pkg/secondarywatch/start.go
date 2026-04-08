package secondarywatch

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/tlsprofile"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/klog/v2"
)

type Options struct {
	Client              *runtime.Client
	ResyncPeriod        time.Duration
	Namespace           string
	PrimaryResourceName string
}

// StarterFunc refers to a function that can be called to start watch on secondary resources.
type StarterFunc func(enqueuer runtime.Enqueuer, shutdown context.Context) error

func (s StarterFunc) Start(enqueuer runtime.Enqueuer, shutdown context.Context) error {
	return s(enqueuer, shutdown)
}

// New sets up watch on secondary resources.
// The function returns lister(s) that can be used to query secondary resources
// and a StarterFunc that can be called to start the watch.
func New(options *Options) (lister *Lister, startFunc StarterFunc) {
	option := informers.WithNamespace(options.Namespace)
	factory := informers.NewSharedInformerFactoryWithOptions(options.Client.Kubernetes, options.ResyncPeriod, option)

	deployment := factory.Apps().V1().Deployments()
	daemonset := factory.Apps().V1().DaemonSets()
	pod := factory.Core().V1().Pods()
	configmap := factory.Core().V1().ConfigMaps()
	service := factory.Core().V1().Services()
	secret := factory.Core().V1().Secrets()
	serviceaccount := factory.Core().V1().ServiceAccounts()
	webhook := factory.Admissionregistration().V1().MutatingWebhookConfigurations()

	apiServerConfigFactory := dynamicinformer.NewDynamicSharedInformerFactory(options.Client.RawDynamic, options.ResyncPeriod)
	apiServerConfigInformer := apiServerConfigFactory.ForResource(tlsprofile.APIServerGVR).Informer()

	startFunc = func(enqueuer runtime.Enqueuer, shutdown context.Context) error {
		handler := newResourceEventHandler(enqueuer)

		deployment.Informer().AddEventHandler(handler)
		daemonset.Informer().AddEventHandler(handler)
		pod.Informer().AddEventHandler(handler)
		configmap.Informer().AddEventHandler(handler)
		service.Informer().AddEventHandler(handler)
		secret.Informer().AddEventHandler(handler)
		serviceaccount.Informer().AddEventHandler(handler)
		webhook.Informer().AddEventHandler(handler)

		if directEnqueuer, ok := enqueuer.(runtime.DirectEnqueuer); ok && options.PrimaryResourceName != "" {
			apiServerConfigInformer.AddEventHandler(
				newDirectEnqueueHandler(directEnqueuer, options.PrimaryResourceName),
			)
		} else {
			klog.Warning("[secondarywatch] enqueuer does not implement DirectEnqueuer or PrimaryResourceName is unset; " +
				"cluster APIServer TLS profile changes will not trigger reconciliation")
		}

		factory.Start(shutdown.Done())
		status := factory.WaitForCacheSync(shutdown.Done())
		if names := check(status); len(names) > 0 {
			return fmt.Errorf("WaitForCacheSync did not successfully complete resources=%s", names)
		}

		apiServerConfigFactory.Start(shutdown.Done())
		if synced := apiServerConfigFactory.WaitForCacheSync(shutdown.Done()); !synced[tlsprofile.APIServerGVR] {
			klog.Warning("[secondarywatch] APIServer config informer cache did not sync; TLS profile watch may be delayed")
		}

		return nil
	}

	lister = &Lister{
		deployment:     deployment.Lister(),
		daemonset:      daemonset.Lister(),
		pod:            pod.Lister(),
		configmap:      configmap.Lister(),
		service:        service.Lister(),
		secret:         secret.Lister(),
		serviceaccount: serviceaccount.Lister(),
		webhook:        webhook.Lister(),
	}

	return
}

func check(status map[reflect.Type]bool) []string {
	names := make([]string, 0)

	for objType, synced := range status {
		if !synced {
			names = append(names, objType.Name())
		}
	}

	return names
}
