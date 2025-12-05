// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"fmt"

	"go.opendefense.cloud/arc/apiserver/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

// Storage is an alias for the apiserver's rest.Storage interface.
// It represents a generic storage backend for Kubernetes resources.
type Storage = rest.Storage

// GetAttrs extracts the labels and fields from a runtime.Object for use in storage predicates.
// Returns an error if the object does not implement resource.Object (i.e., lacks metadata).
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	provider, ok := obj.(resource.Object)
	if !ok {
		return nil, nil, fmt.Errorf("given object of type %T does not have metadata", obj)
	}
	om := provider.GetObjectMeta()
	return om.GetLabels(), SelectableFields(om), nil
}

// SelectableFields returns a set of fields (name, namespace, etc.) for the given ObjectMeta.
// Used for field selectors in storage and API queries.
func SelectableFields(obj *metav1.ObjectMeta) fields.Set {
	return generic.ObjectMetaFieldsSet(obj, true)
}

// NewStore constructs a genericregistry.Store for a Kubernetes resource type.
// It wires up the storage strategies, table conversion, and predicate functions.
//
// Parameters:
//   - scheme: runtime.Scheme for type registration
//   - single: function returning a new instance of the resource
//   - list: function returning a new list instance of the resource
//   - gr: GroupResource describing the resource
//   - strategy: Strategy implementation for create/update/delete/table
//   - optsGetter: RESTOptionsGetter for storage backend configuration
//
// Returns:
//   - *genericregistry.Store: configured store for the resource
//   - error: if store setup fails
func NewStore(
	scheme *runtime.Scheme,
	single, list func() runtime.Object,
	gr schema.GroupResource,
	strategy Strategy, optsGetter generic.RESTOptionsGetter) (*genericregistry.Store, error) {
	store := &genericregistry.Store{
		NewFunc:                   single,
		NewListFunc:               list,
		PredicateFunc:             strategy.Match,
		DefaultQualifiedResource:  gr,
		SingularQualifiedResource: gr,
		TableConvertor:            strategy,
		CreateStrategy:            strategy,
		UpdateStrategy:            strategy,
		DeleteStrategy:            strategy,
	}

	// StoreOptions wires up REST options and attribute extraction for filtering.
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return store, nil
}
