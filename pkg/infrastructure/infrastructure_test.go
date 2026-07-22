package infrastructure

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func makeFakeDynClient(t *testing.T, topology configv1.TopologyMode) *dynamicfake.FakeDynamicClient {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, configv1.AddToScheme(scheme))
	infra := &configv1.Infrastructure{
		TypeMeta:   metav1.TypeMeta{APIVersion: "config.openshift.io/v1", Kind: "Infrastructure"},
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status: configv1.InfrastructureStatus{
			ControlPlaneTopology: topology,
		},
	}
	return dynamicfake.NewSimpleDynamicClient(scheme, infra)
}

func TestIsStandalone(t *testing.T) {
	tests := []struct {
		name     string
		topology configv1.TopologyMode
		want     bool
	}{
		{
			name:     "HighlyAvailable is standalone",
			topology: configv1.HighlyAvailableTopologyMode,
			want:     true,
		},
		{
			name:     "SingleReplica is standalone",
			topology: configv1.SingleReplicaTopologyMode,
			want:     true,
		},
		{
			name:     "External (HCP) is not standalone",
			topology: configv1.ExternalTopologyMode,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dynClient := makeFakeDynClient(t, tt.topology)
			got := IsStandalone(context.Background(), dynClient)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsStandalone_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, configv1.AddToScheme(scheme))
	dynClient := dynamicfake.NewSimpleDynamicClient(scheme)

	// Should default to true (standalone) when Infrastructure CR is missing
	got := IsStandalone(context.Background(), dynClient)
	assert.True(t, got, "should default to standalone when Infrastructure resource is not found")
}
