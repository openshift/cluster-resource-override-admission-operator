package deploy

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
)

func GetDaemonSetStatus(ds *appsv1.DaemonSet) (ready bool, err error) {
	if ds.Generation > ds.Status.ObservedGeneration {
		err = fmt.Errorf("waiting for daemonset spec update name=%s", ds.Name)
		return
	}

	if ds.Status.DesiredNumberScheduled <= 0 || ds.Status.CurrentNumberScheduled <= 0 ||
		ds.Status.DesiredNumberScheduled != ds.Status.CurrentNumberScheduled {
		err = fmt.Errorf("waiting for daemonset pods to be scheduled name=%s", ds.Name)
		return
	}

	if ds.Status.NumberUnavailable > 0 {
		err = fmt.Errorf("waiting for daemonset pods to be available name=%s", ds.Name)
		return
	}

	if ds.Status.DesiredNumberScheduled != ds.Status.NumberAvailable ||
		ds.Status.DesiredNumberScheduled != ds.Status.UpdatedNumberScheduled {
		err = fmt.Errorf("waiting for daemonset pods to be available on all nodes name=%s", ds.Name)
		return
	}

	// DaemonSet is finished
	ready = true
	return
}
