package e2e

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const defaultOperatorNamespace = "openshift-cluster-resource-override"

var namespace = flag.String("namespace", "", "namespace where the operator is deployed; takes precedence over the OPERATOR_NAMESPACE env var")

func init() {
	// Some imported packages (e.g. controller-runtime) register a "kubeconfig"
	// flag in their init(). Avoid a duplicate-registration panic by only
	// registering the flag when it hasn't been registered yet.
	if flag.Lookup("kubeconfig") == nil {
		flag.String("kubeconfig", "", "path to the kubeconfig file")
	}
}

// global test configuration
var options *Options

type Options struct {
	config    *rest.Config
	namespace string
}

func TestMain(m *testing.M) {
	flag.Parse()

	kubeconfig := flag.Lookup("kubeconfig").Value.String()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	// Resolve the operator namespace with the following precedence (highest first):
	//   1. --namespace CLI flag  (always passed by 'make e2e' as --namespace=$(OPERATOR_NAMESPACE))
	//   2. OPERATOR_NAMESPACE environment variable  (used when running the test binary directly)
	//   3. built-in default (openshift-cluster-resource-override)
	ns := *namespace
	if ns == "" {
		ns = os.Getenv("OPERATOR_NAMESPACE")
	}
	if ns == "" {
		ns = defaultOperatorNamespace
	}

	// Validate at the trust boundary before the value reaches any Kubernetes
	// API call or is assigned to shared test state. Use the same IsDNS1123Label
	// function Kubernetes itself uses to validate namespace names — this ensures
	// our check stays in sync with Kubernetes validation rules automatically.
	if errs := validation.IsDNS1123Label(ns); len(errs) > 0 {
		panic(fmt.Sprintf(
			"invalid operator namespace %q: %s",
			ns, strings.Join(errs, "; "),
		))
	}

	// Expose the resolved namespace through both options.namespace (used by
	// TestDynamicClient) and the package-level operatorNamespace variable (used
	// by every other test that references the operator deployment directly).
	operatorNamespace = ns

	options = &Options{
		config:    config,
		namespace: ns,
	}

	// run tests
	os.Exit(m.Run())
}
