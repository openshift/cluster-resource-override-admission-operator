package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (in *PodResourceOverrideSpec) String() string {
	return fmt.Sprintf("ForceSelinuxRelabel=%t, MemoryRequestToLimitPercent=%d, CPURequestToLimitPercent=%d, LimitCPUToMemoryPercent=%d, CPURequestToRequestPercent=%d",
		in.ForceSelinuxRelabel, in.MemoryRequestToLimitPercent, in.CPURequestToLimitPercent, in.LimitCPUToMemoryPercent, in.CPURequestToRequestPercent)
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

	if in.CPURequestToRequestPercent < 0 || in.CPURequestToRequestPercent > 100 {
		return errors.New("invalid value for CPURequestToRequestPercent, must be [0...100]")
	}

	return nil
}

func (in *PodResourceOverrideSpec) Hash() string {
	value := in.String()

	writer := sha256.New()
	writer.Write([]byte(value))
	return hex.EncodeToString(writer.Sum(nil))
}

func (in *ResourceOverrideSpec) Hash() string {
	value := fmt.Sprintf("PodResourceOverride=%s, PodSelector=%s", in.PodResourceOverride.Hash(), hashLabelSelector(in.PodSelector))

	writer := sha256.New()
	writer.Write([]byte(value))
	return hex.EncodeToString(writer.Sum(nil))
}

func hashLabelSelector(sel *metav1.LabelSelector) string {
	if sel == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("MatchLabels=")
	sb.WriteString(mapToString(sel.MatchLabels))
	sb.WriteString(",MatchExpressions=")
	exprs := make([]string, len(sel.MatchExpressions))
	for i, e := range sel.MatchExpressions {
		exprs[i] = fmt.Sprintf("%s %s [%s]", e.Key, e.Operator, strings.Join(e.Values, ","))
	}
	sort.Strings(exprs)
	sb.WriteString(strings.Join(exprs, ";"))
	return sb.String()
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
