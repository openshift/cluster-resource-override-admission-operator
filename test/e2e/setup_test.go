package e2e

import (
	"flag"
	"os"
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var namespace = flag.String("namespace", "", "namespace where tests will run")

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

	options = &Options{
		config:    config,
		namespace: *namespace,
	}

	// run tests
	os.Exit(m.Run())
}
