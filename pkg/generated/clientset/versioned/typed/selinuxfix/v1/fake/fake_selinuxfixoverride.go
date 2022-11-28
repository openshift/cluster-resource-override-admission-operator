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

package fake

import (
	"context"

	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSelinuxFixOverrides implements SelinuxFixOverrideInterface
type FakeSelinuxFixOverrides struct {
	Fake *FakeSelinuxfixV1
}

var selinuxfixoverridesResource = schema.GroupVersionResource{Group: "selinuxfix.node.openshift.io", Version: "v1", Resource: "selinuxfixoverrides"}

var selinuxfixoverridesKind = schema.GroupVersionKind{Group: "selinuxfix.node.openshift.io", Version: "v1", Kind: "SelinuxFixOverride"}

// Get takes name of the selinuxFixOverride, and returns the corresponding selinuxFixOverride object, and an error if there is any.
func (c *FakeSelinuxFixOverrides) Get(ctx context.Context, name string, options v1.GetOptions) (result *selinuxfixv1.SelinuxFixOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(selinuxfixoverridesResource, name), &selinuxfixv1.SelinuxFixOverride{})
	if obj == nil {
		return nil, err
	}
	return obj.(*selinuxfixv1.SelinuxFixOverride), err
}

// List takes label and field selectors, and returns the list of SelinuxFixOverrides that match those selectors.
func (c *FakeSelinuxFixOverrides) List(ctx context.Context, opts v1.ListOptions) (result *selinuxfixv1.SelinuxFixOverrideList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(selinuxfixoverridesResource, selinuxfixoverridesKind, opts), &selinuxfixv1.SelinuxFixOverrideList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &selinuxfixv1.SelinuxFixOverrideList{ListMeta: obj.(*selinuxfixv1.SelinuxFixOverrideList).ListMeta}
	for _, item := range obj.(*selinuxfixv1.SelinuxFixOverrideList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested selinuxFixOverrides.
func (c *FakeSelinuxFixOverrides) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(selinuxfixoverridesResource, opts))
}

// Create takes the representation of a selinuxFixOverride and creates it.  Returns the server's representation of the selinuxFixOverride, and an error, if there is any.
func (c *FakeSelinuxFixOverrides) Create(ctx context.Context, selinuxFixOverride *selinuxfixv1.SelinuxFixOverride, opts v1.CreateOptions) (result *selinuxfixv1.SelinuxFixOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(selinuxfixoverridesResource, selinuxFixOverride), &selinuxfixv1.SelinuxFixOverride{})
	if obj == nil {
		return nil, err
	}
	return obj.(*selinuxfixv1.SelinuxFixOverride), err
}

// Update takes the representation of a selinuxFixOverride and updates it. Returns the server's representation of the selinuxFixOverride, and an error, if there is any.
func (c *FakeSelinuxFixOverrides) Update(ctx context.Context, selinuxFixOverride *selinuxfixv1.SelinuxFixOverride, opts v1.UpdateOptions) (result *selinuxfixv1.SelinuxFixOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(selinuxfixoverridesResource, selinuxFixOverride), &selinuxfixv1.SelinuxFixOverride{})
	if obj == nil {
		return nil, err
	}
	return obj.(*selinuxfixv1.SelinuxFixOverride), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeSelinuxFixOverrides) UpdateStatus(ctx context.Context, selinuxFixOverride *selinuxfixv1.SelinuxFixOverride, opts v1.UpdateOptions) (*selinuxfixv1.SelinuxFixOverride, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(selinuxfixoverridesResource, "status", selinuxFixOverride), &selinuxfixv1.SelinuxFixOverride{})
	if obj == nil {
		return nil, err
	}
	return obj.(*selinuxfixv1.SelinuxFixOverride), err
}

// Delete takes name of the selinuxFixOverride and deletes it. Returns an error if one occurs.
func (c *FakeSelinuxFixOverrides) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(selinuxfixoverridesResource, name), &selinuxfixv1.SelinuxFixOverride{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSelinuxFixOverrides) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(selinuxfixoverridesResource, listOpts)

	_, err := c.Fake.Invokes(action, &selinuxfixv1.SelinuxFixOverrideList{})
	return err
}

// Patch applies the patch and returns the patched selinuxFixOverride.
func (c *FakeSelinuxFixOverrides) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *selinuxfixv1.SelinuxFixOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(selinuxfixoverridesResource, name, pt, data, subresources...), &selinuxfixv1.SelinuxFixOverride{})
	if obj == nil {
		return nil, err
	}
	return obj.(*selinuxfixv1.SelinuxFixOverride), err
}
