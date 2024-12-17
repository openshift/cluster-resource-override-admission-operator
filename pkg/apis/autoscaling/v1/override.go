package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func (in *PodResourceOverrideSpec) String() string {
	return fmt.Sprintf("MemoryRequestToLimitPercent=%d, CPURequestToLimitPercent=%d, LimitCPUToMemoryPercent=%d, ForceSelinuxRelabel=%t",
		in.MemoryRequestToLimitPercent, in.CPURequestToLimitPercent, in.LimitCPUToMemoryPercent, in.ForceSelinuxRelabel)
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

func (in *DeploymentOverrides) String() string {
	replicas := "nil"
	if in.Replicas != nil {
		replicas = fmt.Sprintf("%v", *in.Replicas)
	}
	return fmt.Sprintf("Replicas=%s, NodeSelector=%s, Tolerations=%s", replicas, mapToString(in.NodeSelector), tolerationsToString(in.Tolerations))
}

func (in *DeploymentOverrides) Validate() error {
	if in.Replicas != nil && *in.Replicas < 0 {
		return errors.New("invalid value for Replicas, must be a positive value")
	}

	return nil
}

func (in *DeploymentOverrides) Hash() string {
	value := in.String()

	writer := sha256.New()
	writer.Write([]byte(value))
	return hex.EncodeToString(writer.Sum(nil))
}

func (in *ClusterResourceOverrideSpec) Hash() string {
	value := fmt.Sprintf("PodResourceOverride=%s, DeploymentOverrides=%s", in.PodResourceOverride.Spec.Hash(), in.DeploymentOverrides.Hash())

	writer := sha256.New()
	writer.Write([]byte(value))
	return hex.EncodeToString(writer.Sum(nil))
}

func mapToString(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%s=%s,", k, m[k]))
	}
	return sb.String()
}

func tolerationsToString(tolerations []corev1.Toleration) string {
	return fmt.Sprintf("%v", tolerations)
}
