package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/cluster-resource-override-admission-operator/test/helper"
)

// TestTLSScannerPQC runs the tls-scanner tool against the webhook service
// endpoint with --pqc-check to verify post-quantum cryptography readiness.
func TestTLSScannerPQC(t *testing.T) {
	tlsScannerBin := os.Getenv("TLS_SCANNER_BIN")
	if tlsScannerBin == "" {
		t.Skip("TLS_SCANNER_BIN not set, skipping tls-scanner test")
	}

	if _, err := os.Stat(tlsScannerBin); err != nil {
		t.Skipf("tls-scanner binary not found at %s: %v", tlsScannerBin, err)
	}

	client := helper.NewClient(t, options.config)
	ensureOperandDeployment(t, client)

	// Get the webhook service ClusterIP.
	svc, err := client.Kubernetes.CoreV1().Services(operatorNamespace).Get(
		context.TODO(), operandName, metav1.GetOptions{},
	)
	require.NoError(t, err, "failed to get webhook service")
	require.NotEmpty(t, svc.Spec.ClusterIP, "webhook service has no ClusterIP")

	host := svc.Spec.ClusterIP
	port := "443"
	for _, p := range svc.Spec.Ports {
		if p.Port != 0 {
			port = fmt.Sprintf("%d", p.Port)
			break
		}
	}

	t.Logf("running tls-scanner against %s:%s", host, port)

	cmd := exec.Command(tlsScannerBin, "--host", host, "--port", port, "--pqc-check")
	output, err := cmd.CombinedOutput()
	t.Logf("tls-scanner output:\n%s", string(output))
	require.NoError(t, err, "tls-scanner exited with error")
}
