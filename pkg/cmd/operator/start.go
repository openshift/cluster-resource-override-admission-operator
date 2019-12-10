package operator

import (
	"fmt"
	"github.com/spf13/cobra"
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
			_, err := load(cmd)
			if err != nil {
				klog.Errorf("error loading configuration - %s", err.Error())
				os.Exit(1)
			}
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
