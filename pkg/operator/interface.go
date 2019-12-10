package operator

import (
	"context"
	"errors"
	"fmt"
	"k8s.io/client-go/rest"
)

// Interface is the interface that defines an operator.
type Interface interface {
	// Run will start the operator instance given a config.
	Run(config *Config, errorCh chan<- error)

	// Done returns a channel that's closed when the operator is done.
	Done() <-chan struct{}
}

type Config struct {
	// Name is the name of the operator. This name will be used to create kube resources.
	// More info: http://kubernetes.io/docs/user-guide/identifiers#names.
	Name string

	// Namespace is the namespace where the operator is installed.
	Namespace string

	// ShutdownContext is the parent context.
	ShutdownContext context.Context

	// RestConfig is the rest.Config object to be used to build clients.
	RestConfig *rest.Config

	// OperandImage points to operand image.
	OperandImage string

	// OperandVersion points to operand version.
	OperandVersion string
}

func (c *Config) String() string {
	return fmt.Sprintf("name=%s namespace=%s operand-image=%s operand-version=%s", c.Name, c.Namespace, c.OperandImage, c.OperandVersion)
}

func (c *Config) Validate() error {
	if c.Namespace == "" {
		return errors.New("operator namespace must be specified")
	}

	if c.Name == "" {
		return errors.New("operator name must be specified")
	}

	if c.RestConfig == nil {
		return errors.New("no rest.Config has been specified")
	}

	if c.OperandImage == "" {
		return errors.New("no operand image has been specified")
	}

	if c.OperandVersion == "" {
		return errors.New("no operand version has been specified")
	}

	return nil
}
