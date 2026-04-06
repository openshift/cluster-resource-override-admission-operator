package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/tlsprofile"
	"github.com/openshift/cluster-resource-override-admission-operator/test/helper"
)

const operandName = "clusterresourceoverride"

// verifyOperandTLSArgs fetches the current cluster APIServer TLS profile and
// waits until the operand deployment's container args reflect it.
func verifyOperandTLSArgs(t *testing.T, kubeClient *helper.Client, config *rest.Config) {
	t.Helper()

	dynClient, err := dynamic.NewForConfig(config)
	require.NoError(t, err)

	tlsArgs, err := tlsprofile.Fetch(context.TODO(), dynClient)
	require.NoError(t, err)

	helper.WaitForOperandTLSArgs(t, kubeClient.Kubernetes, operatorNamespace, operandName, operandName, tlsArgs)
}

// TestOperandTLSProfileArgs verifies that the operand deployment's --tls-min-version
// and --tls-cipher-suites args always match the cluster APIServer TLS profile across
// three scenarios: initial config, a custom profile, and restoration of the original.
func TestOperandTLSProfileArgs(t *testing.T) {
	client := helper.NewClient(t, options.config)

	// Ensure the CRO CR and operand deployment exist before checking TLS args.
	override := autoscalingv1.PodResourceOverride{
		Spec: autoscalingv1.PodResourceOverrideSpec{
			MemoryRequestToLimitPercent: 50,
			CPURequestToLimitPercent:    25,
			LimitCPUToMemoryPercent:     200,
		},
	}
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())
	t.Log("waiting for operand to become available")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))
	_ = current

	// Save the initial APIServer TLS profile so we can restore it after the test.
	initialProfile := helper.GetAPIServerTLSProfile(t, options.config)
	t.Cleanup(func() {
		helper.SetAPIServerTLSProfile(t, options.config, initialProfile)
	})

	// Scenario 1: verify TLS args match the initial cluster configuration.
	t.Log("scenario 1: verifying operand TLS args match initial cluster APIServer TLS profile")
	verifyOperandTLSArgs(t, client, options.config)

	// Scenario 2: change the APIServer TLS profile to a custom profile.
	t.Log("scenario 2: changing cluster APIServer TLS profile to custom profile")
	customProfile := []byte(`{"type":"Custom","custom":{"ciphers":["ECDHE-ECDSA-CHACHA20-POLY1305","ECDHE-ECDSA-AES128-GCM-SHA256"]}}`)
	helper.SetAPIServerTLSProfile(t, options.config, customProfile)
	verifyOperandTLSArgs(t, client, options.config)

	// Scenario 3: revert to the initial cluster configuration.
	t.Log("scenario 3: reverting cluster APIServer TLS profile to initial configuration")
	helper.SetAPIServerTLSProfile(t, options.config, initialProfile)
	verifyOperandTLSArgs(t, client, options.config)
}
