package e2e

import (
	"flag"
	"k8s.io/client-go/rest"
	"os"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeConfigPath = flag.String(
		"kubeconfig", "", "path to the kubeconfig file")

	namespace = flag.String(
		"namespace", "", "namespace where tests will run")
)

// global test configuration
var options *Options

type Options struct {
	config    *rest.Config
	namespace string
}

func TestMain(m *testing.M) {
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
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
