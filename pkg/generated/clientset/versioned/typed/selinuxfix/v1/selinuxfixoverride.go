/*
Copyright 2022 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	v1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	scheme "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SelinuxFixOverridesGetter has a method to return a SelinuxFixOverrideInterface.
// A group's client should implement this interface.
type SelinuxFixOverridesGetter interface {
	SelinuxFixOverrides() SelinuxFixOverrideInterface
}

// SelinuxFixOverrideInterface has methods to work with SelinuxFixOverride resources.
type SelinuxFixOverrideInterface interface {
	Create(ctx context.Context, selinuxFixOverride *v1.SelinuxFixOverride, opts metav1.CreateOptions) (*v1.SelinuxFixOverride, error)
	Update(ctx context.Context, selinuxFixOverride *v1.SelinuxFixOverride, opts metav1.UpdateOptions) (*v1.SelinuxFixOverride, error)
	UpdateStatus(ctx context.Context, selinuxFixOverride *v1.SelinuxFixOverride, opts metav1.UpdateOptions) (*v1.SelinuxFixOverride, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.SelinuxFixOverride, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.SelinuxFixOverrideList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.SelinuxFixOverride, err error)
	SelinuxFixOverrideExpansion
}

// selinuxFixOverrides implements SelinuxFixOverrideInterface
type selinuxFixOverrides struct {
	client rest.Interface
}

// newSelinuxFixOverrides returns a SelinuxFixOverrides
func newSelinuxFixOverrides(c *SelinuxfixV1Client) *selinuxFixOverrides {
	return &selinuxFixOverrides{
		client: c.RESTClient(),
	}
}

// Get takes name of the selinuxFixOverride, and returns the corresponding selinuxFixOverride object, and an error if there is any.
func (c *selinuxFixOverrides) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.SelinuxFixOverride, err error) {
	result = &v1.SelinuxFixOverride{}
	err = c.client.Get().
		Resource("selinuxfixoverrides").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SelinuxFixOverrides that match those selectors.
func (c *selinuxFixOverrides) List(ctx context.Context, opts metav1.ListOptions) (result *v1.SelinuxFixOverrideList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.SelinuxFixOverrideList{}
	err = c.client.Get().
		Resource("selinuxfixoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested selinuxFixOverrides.
func (c *selinuxFixOverrides) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("selinuxfixoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a selinuxFixOverride and creates it.  Returns the server's representation of the selinuxFixOverride, and an error, if there is any.
func (c *selinuxFixOverrides) Create(ctx context.Context, selinuxFixOverride *v1.SelinuxFixOverride, opts metav1.CreateOptions) (result *v1.SelinuxFixOverride, err error) {
	result = &v1.SelinuxFixOverride{}
	err = c.client.Post().
		Resource("selinuxfixoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(selinuxFixOverride).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a selinuxFixOverride and updates it. Returns the server's representation of the selinuxFixOverride, and an error, if there is any.
func (c *selinuxFixOverrides) Update(ctx context.Context, selinuxFixOverride *v1.SelinuxFixOverride, opts metav1.UpdateOptions) (result *v1.SelinuxFixOverride, err error) {
	result = &v1.SelinuxFixOverride{}
	err = c.client.Put().
		Resource("selinuxfixoverrides").
		Name(selinuxFixOverride.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(selinuxFixOverride).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *selinuxFixOverrides) UpdateStatus(ctx context.Context, selinuxFixOverride *v1.SelinuxFixOverride, opts metav1.UpdateOptions) (result *v1.SelinuxFixOverride, err error) {
	result = &v1.SelinuxFixOverride{}
	err = c.client.Put().
		Resource("selinuxfixoverrides").
		Name(selinuxFixOverride.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(selinuxFixOverride).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the selinuxFixOverride and deletes it. Returns an error if one occurs.
func (c *selinuxFixOverrides) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("selinuxfixoverrides").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *selinuxFixOverrides) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("selinuxfixoverrides").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched selinuxFixOverride.
func (c *selinuxFixOverrides) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.SelinuxFixOverride, err error) {
	result = &v1.SelinuxFixOverride{}
	err = c.client.Patch(pt).
		Resource("selinuxfixoverrides").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
