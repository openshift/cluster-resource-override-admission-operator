package e2e

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	operatorv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/operator/v1"
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

// ensureOperandDeployment creates the CRO CR if needed and waits for the operand
// deployment to become available. Returns a cleanup function that removes the CR.
func ensureOperandDeployment(t *testing.T, client *helper.Client) {
	t.Helper()

	override := operatorv1.PodResourceOverride{
		Spec: operatorv1.PodResourceOverrideSpec{
			MemoryRequestToLimitPercent: 50,
			CPURequestToLimitPercent:    25,
			LimitCPUToMemoryPercent:     200,
		},
	}
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	t.Cleanup(func() { helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName()) })
	t.Log("waiting for operand to become available")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))
	_ = current
}

// TestOperandTLSProfileArgs verifies that the operand deployment's --tls-min-version
// and --tls-cipher-suites args always match the cluster APIServer TLS profile across
// three scenarios: initial config, a custom profile, and restoration of the original.
// The test sets StrictAllComponents so that TLS profile changes are reflected in the operand.
func TestOperandTLSProfileArgs(t *testing.T) {
	client := helper.NewClient(t, options.config)
	ensureOperandDeployment(t, client)

	// Save the initial APIServer TLS adherence policy and profile so we can restore them.
	initialAdherence := helper.GetAPIServerTLSAdherencePolicy(t, options.config)
	initialProfile := helper.GetAPIServerTLSProfile(t, options.config)
	t.Cleanup(func() {
		helper.SetAPIServerTLSAdherencePolicy(t, options.config, initialAdherence)
		helper.SetAPIServerTLSProfile(t, options.config, initialProfile)
	})

	// Require StrictAllComponents so that TLS profile changes are reflected in the operand.
	// Skip the test if the TLSAdherence feature gate is not available on this cluster.
	if err := helper.TrySetAPIServerTLSAdherencePolicy(t, options.config, string(configv1.TLSAdherencePolicyStrictAllComponents)); err != nil {
		t.Skipf("skipping: TLSAdherence feature gate not available on this cluster: %v", err)
	}

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

// TestOperandTLSAdherencePolicy verifies that the operand deployment's TLS args
// respond correctly to changes in the cluster APIServer TLSAdherence policy.
// Requires the TLSAdherence feature gate to be enabled; skips otherwise.
func TestOperandTLSAdherencePolicy(t *testing.T) {
	client := helper.NewClient(t, options.config)
	ensureOperandDeployment(t, client)

	// Save the initial state and restore on cleanup.
	initialAdherence := helper.GetAPIServerTLSAdherencePolicy(t, options.config)
	initialProfile := helper.GetAPIServerTLSProfile(t, options.config)
	t.Cleanup(func() {
		helper.SetAPIServerTLSAdherencePolicy(t, options.config, initialAdherence)
		helper.SetAPIServerTLSProfile(t, options.config, initialProfile)
	})

	// Set a known TLS profile for the duration of this test so we can verify
	// that the operand picks it up when StrictAllComponents is active.
	customProfile := []byte(`{"type":"Custom","custom":{"ciphers":["ECDHE-ECDSA-CHACHA20-POLY1305","ECDHE-ECDSA-AES128-GCM-SHA256"]}}`)
	helper.SetAPIServerTLSProfile(t, options.config, customProfile)

	// Scenario A: NoOpinion (field cleared) — operand should use its own defaults (no TLS flags).
	// This should also succeed if the feature gate is off (though possibly with a warning about a missing field)
	t.Log("scenario A: NoOpinion — operand should use its own defaults")
	helper.SetAPIServerTLSAdherencePolicy(t, options.config, "")
	verifyOperandTLSArgs(t, client, options.config)

	// Scenarios B and C require the TLSAdherence feature gate.
	if !helper.IsFeatureGateEnabled(t, options.config, "TLSAdherence") {
		t.Skip("skipping scenarios B and C: TLSAdherence feature gate is not enabled")
	}

	// Scenario B: LegacyAdheringComponentsOnly — operand should use its own defaults (no TLS flags).
	t.Log("scenario B: LegacyAdheringComponentsOnly — operand should use its own defaults")
	helper.SetAPIServerTLSAdherencePolicy(t, options.config, string(configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly))
	verifyOperandTLSArgs(t, client, options.config)

	// Scenario C: StrictAllComponents — operand must apply the cluster TLS profile.
	t.Log("scenario C: StrictAllComponents — operand should apply cluster TLS profile")
	helper.SetAPIServerTLSAdherencePolicy(t, options.config, string(configv1.TLSAdherencePolicyStrictAllComponents))
	verifyOperandTLSArgs(t, client, options.config)
}
