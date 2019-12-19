package controller

import (
	"context"
	"errors"
	"fmt"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// NewRunner returns a new instance of Runner.
func NewRunner() Runner {
	return &runner{
		worker: Work,
		done:   make(chan struct{}, 0),
	}
}

type runner struct {
	done   chan struct{}
	worker WorkerFunc
}

func (r *runner) Run(parent context.Context, controller Interface, errorCh chan<- error) {
	defer func() {
		close(r.done)
	}()

	if parent == nil || controller == nil {
		errorCh <- errors.New("invalid input to Runner.Run")
		return
	}

	defer utilruntime.HandleCrash()
	defer controller.Queue().ShutDown()

	klog.V(1).Infof("[controller] name=%s starting informer", controller.Name())
	go controller.Informer().Run(parent.Done())

	klog.V(1).Infof("[controller] name=%s waiting for informer cache to sync", controller.Name())
	if ok := cache.WaitForCacheSync(parent.Done(), controller.Informer().HasSynced); !ok {
		errorCh <- fmt.Errorf("controller=%s failed to wait for caches to sync", controller.Name())
		return
	}

	for i := 0; i < controller.WorkerCount(); i++ {
		go r.worker.Work(parent, controller)
	}

	klog.V(1).Infof("[controller] name=%s started %d worker(s)", controller.Name(), controller.WorkerCount())
	errorCh <- nil
	klog.V(1).Infof("[controller] name=%s waiting ", controller.Name())

	// Not waiting for any child to finish, waiting for the parent to signal done.
	<-parent.Done()

	klog.V(1).Infof("[controller] name=%s shutting down queue", controller.Name())
}

func (r *runner) Done() <-chan struct{} {
	return r.done
}
