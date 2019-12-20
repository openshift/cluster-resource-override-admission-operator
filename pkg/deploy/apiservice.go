package deploy

import (
	corev1 "k8s.io/api/core/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

func IsAPIServiceAvailable(apiservice *apiregistrationv1.APIService) (status corev1.ConditionStatus, message string) {
	cond := getAvailableCondition(apiservice)
	if cond == nil {
		status = corev1.ConditionFalse
		return
	}

	switch cond.Status {
	case apiregistrationv1.ConditionTrue:
		status = corev1.ConditionTrue
	default:
		status = corev1.ConditionFalse
	}

	message = cond.Message
	return
}

func getAvailableCondition(apiservice *apiregistrationv1.APIService) *apiregistrationv1.APIServiceCondition {
	for i := range apiservice.Status.Conditions {
		c := &apiservice.Status.Conditions[i]
		if c.Type == apiregistrationv1.Available {
			return c
		}
	}

	return nil
}
