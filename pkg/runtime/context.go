package runtime

const (
	Operator = "operator.clusteroverride.openshift.io"
)

func NewOperandContext(name, namespace, resource, image, version string) OperandContext {
	return &context{
		name:      name,
		namespace: namespace,
		resource:  resource,
		image:     image,
		version:   version,
	}
}

type OperandContext interface {
	// Name is the name of the ClusterResourceOverride admission webhook server.
	// This name will be used to create kube resources.
	// // More info: http://kubernetes.io/docs/user-guide/identifiers#names.
	WebhookName() string

	// Namespace is the namespace where the ClusterResourceOverride admission
	// webhook server is installed.
	WebhookNamespace() string

	// OperandImage points to the operand (ClusterResourceOverride admission webhook) image.
	OperandImage() string

	// OperandVersion is the version of the operand (ClusterResourceOverride admission webhook).
	OperandVersion() string

	// ResourceName is the name of the CustomResource that will manage this operand.
	ResourceName() string
}

type context struct {
	name      string
	namespace string
	resource  string
	image     string
	version   string
}

func (c *context) WebhookName() string {
	return c.name
}

func (c *context) WebhookNamespace() string {
	return c.namespace
}

func (c *context) OperandImage() string {
	return c.image
}

func (c *context) OperandVersion() string {
	return c.version
}

func (c *context) ResourceName() string {
	return c.resource
}
