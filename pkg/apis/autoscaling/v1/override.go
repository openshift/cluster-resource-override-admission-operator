package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (in *ClusterResourceOverride) IsTimeToRotateCert() bool {
	if in.Status.CertsRotateAt.IsZero() {
		return true
	}

	now := metav1.Now()
	if in.Status.CertsRotateAt.Before(&now) {
		return true
	}

	return false
}

func (in *PodResourceOverrideSpec) String() string {
	return fmt.Sprintf("MemoryRequestToLimitPercent=%d, CPURequestToLimitPercent=%d LimitCPUToMemoryPercent=%d",
		in.MemoryRequestToLimitPercent, in.CPURequestToLimitPercent, in.LimitCPUToMemoryPercent)
}

func (in *PodResourceOverrideSpec) Validate() error {
	if in.MemoryRequestToLimitPercent < 0 || in.MemoryRequestToLimitPercent > 100 {
		return errors.New("invalid value for MemoryRequestToLimitPercent, must be [0...100]")
	}

	if in.CPURequestToLimitPercent < 0 || in.CPURequestToLimitPercent > 100 {
		return errors.New("invalid value for CPURequestToLimitPercent, must be [0...100]")
	}

	if in.LimitCPUToMemoryPercent < 0 {
		return errors.New("invalid value for LimitCPUToMemoryPercent, must be a positive value")
	}

	return nil
}

func (in *PodResourceOverrideSpec) Hash() string {
	value := in.String()

	writer := sha256.New()
	writer.Write([]byte(value))
	return hex.EncodeToString(writer.Sum(nil))
}

func (in *DeploymentOverrides) Validate() error {
	if in.Replicas != nil && *in.Replicas < 0 {
		return errors.New("invalid value for Replicas, must be a positive value")
	}

	return nil
}
