package operator

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"os"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/operator"
)

const (
	OperatorName          = "clusterresourceoverride"
	OperandImageEnvName   = "OPERAND_IMAGE"
	OperandVersionEnvName = "OPERAND_VERSION"
)

func NewStartCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "start",
		Short: "Start the operator",
		Long:  `starts launches the operator in the foreground.`,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := load(cmd)
			if err != nil {
				klog.Errorf("error loading configuration - %s", err.Error())
				os.Exit(1)
			}

			run(config)
		},
	}

	command.Flags().String("kubeconfig", "", "absolute path to kubeconfig file")
	command.Flags().String("namespace", "", "operator namespace")

	return command
}

func load(command *cobra.Command) (config *operator.Config, err error) {
	kubeconfig, err := command.Flags().GetString("kubeconfig")
	if err != nil {
		return
	}

	namespace, err := command.Flags().GetString("namespace")
	if err != nil {
		return
	}

	operandImage := os.Getenv(OperandImageEnvName)
	if operandImage == "" {
		err = fmt.Errorf("%s=<empty> no operand image specified", OperandImageEnvName)
		return
	}

	operandVersion := os.Getenv(OperandVersionEnvName)
	if operandVersion == "" {
		err = fmt.Errorf("%s=<empty> no operand version specified", OperandVersionEnvName)
		return
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		err = fmt.Errorf("error building kubeconfig: %s", err.Error())
		return
	}

	c := &operator.Config{
		Namespace:      namespace,
		Name:           OperatorName,
		RestConfig:     restConfig,
		OperandImage:   operandImage,
		OperandVersion: operandVersion,
	}
	if validationError := c.Validate(); validationError != nil {
		err = fmt.Errorf("invalid configuration: %s", validationError.Error())
		return
	}

	config = c
	return
}

func run(config *operator.Config) {
	shutdown, cancel := context.WithCancel(context.TODO())
	config.ShutdownContext = shutdown

	shutdownHandler := server.SetupSignalHandler()
	go func() {
		defer cancel()

		<-shutdownHandler
		klog.V(1).Info("[operator] Received SIGTERM or SIGINT signal, initiating shutdown.")
	}()

	klog.V(1).Infof("[operator] configuration - %s", config)
	klog.V(1).Info("[operator] starting")

	errorCh := make(chan error, 0)
	runner := operator.NewRunner()
	go runner.Run(config, errorCh)
	if err := <-errorCh; err != nil {
		klog.V(1).Infof("error running operator - %s", err.Error())
	}

	klog.Infof("[operator] operator is running, waiting for the operator to be done.")
	<-runner.Done()
	klog.Infof("process exiting.")
}
