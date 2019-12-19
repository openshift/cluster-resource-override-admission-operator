package deploy

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	TimedOutReason = "ProgressDeadlineExceeded"
)

func GetDeploymentCondition(status *appsv1.DeploymentStatus, condType appsv1.DeploymentConditionType) (condition *appsv1.DeploymentCondition) {
	for i := range status.Conditions {
		if condType != status.Conditions[i].Type {
			continue
		}

		condition = &status.Conditions[i]
		return
	}

	return
}

func IsDeploymentFailedCreate(status *appsv1.DeploymentStatus) bool {
	cond := GetDeploymentCondition(status, appsv1.DeploymentReplicaFailure)
	if cond == nil {
		return false
	}

	return cond.Reason == "FailedCreate" && cond.Status == corev1.ConditionTrue
}

func GetDeploymentStatus(deployment *appsv1.Deployment) (done bool, err error) {
	if deployment.Generation > deployment.Status.ObservedGeneration {
		err = fmt.Errorf("waiting for deployment spec update name=%s", deployment.Name)
		return
	}

	condition := GetDeploymentCondition(&deployment.Status, appsv1.DeploymentProgressing)
	if condition != nil && condition.Reason == TimedOutReason {
		err = fmt.Errorf("deployment exceeded its progress deadline name=%s", deployment.Name)
		return
	}

	// not all replicas are up yet
	if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
		err = fmt.Errorf("waiting for rollout to finish: %d out of %d new replicas have been updated",
			deployment.Status.UpdatedReplicas, *deployment.Spec.Replicas)
		return
	}

	// waiting for old replicas to be cleaned up
	if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
		err = fmt.Errorf("waiting for rollout to finish: %d old replicas are pending termination", deployment.Status.Replicas-deployment.Status.UpdatedReplicas)
		return
	}

	// waiting for new replicas to report as available
	if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
		err = fmt.Errorf("waiting for rollout to finish: %d of %d updated replicas are available", deployment.Status.AvailableReplicas, deployment.Status.UpdatedReplicas)
		return
	}

	// deployment is finished
	done = true
	return
}
