package operator

import (
	"fmt"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
	"k8s.io/klog"
	"net/http"
	"time"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/controller"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
)

const (
	// DefaultCR is the name of the global cluster-scoped ClusterResourceOverride object that
	// the operator will be watching for.
	// All other ClusterResourceOverride object(s) are ignored since the operand is
	// basically a cluster singleton.
	DefaultCR = "cluster"

	// Default worker count is 1.
	DefaultWorkerCount = 1

	// Default ResyncPeriod for primary (ClusterResourceOverride objects)
	DefaultResyncPeriodPrimaryResource = 1 * time.Hour

	// Default ResyncPeriod for all secondary resources that the operator manages.
	DefaultResyncPeriodSecondaryResource = 15 * time.Hour
)

func NewRunner() Interface {
	return &runner{
		done: make(chan struct{}, 0),
	}
}

type runner struct {
	done chan struct{}
}

func (r *runner) Run(config *Config, errorCh chan<- error) {
	defer func() {
		close(r.done)
		klog.V(1).Infof("[operator] exiting")
	}()

	clients, err := runtime.NewClient(config.RestConfig)
	if err != nil {
		errorCh <- err
		return
	}

	context := runtime.NewOperandContext(config.Name, config.Namespace, DefaultCR, config.OperandImage, config.OperandVersion)

	// create lister(s) for secondary resources
	lister, starter := secondarywatch.New(&secondarywatch.Options{
		Client:       clients,
		ResyncPeriod: DefaultResyncPeriodSecondaryResource,
		Namespace:    config.Namespace,
	})

	// start the controllers
	c, enqueuer, err := clusterresourceoverride.New(&clusterresourceoverride.Options{
		ResyncPeriod:   DefaultResyncPeriodPrimaryResource,
		Workers:        DefaultWorkerCount,
		RuntimeContext: context,
		Client:         clients,
		Lister:         lister,
	})
	if err != nil {
		errorCh <- fmt.Errorf("failed to create controller - %s", err.Error())
		return
	}

	// setup watches for secondary resources
	if err := starter.Start(enqueuer, config.ShutdownContext); err != nil {
		errorCh <- fmt.Errorf("failed to start watch on secondary resources - %s", err.Error())
		return
	}

	runner := controller.NewRunner()
	runnerErrorCh := make(chan error, 0)
	go runner.Run(config.ShutdownContext, c, runnerErrorCh)
	if err := <-runnerErrorCh; err != nil {
		errorCh <- err
		return
	}

	// Serve a simple HTTP health check.
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(":8080", healthMux)

	errorCh <- nil
	klog.V(1).Infof("operator is waiting for controller(s) to be done")

	<-runner.Done()
}

func (r *runner) Done() <-chan struct{} {
	return r.done
}
