// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package order

import (
	"go.opendefense.cloud/arc/api/arc"
	"go.opendefense.cloud/arc/pkg/registry"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (*registry.REST, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc:                   func() runtime.Object { return &arc.Order{} },
		NewListFunc:               func() runtime.Object { return &arc.OrderList{} },
		PredicateFunc:             MatchOrder,
		DefaultQualifiedResource:  arc.Resource("orders"),
		SingularQualifiedResource: arc.Resource("order"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		// TODO: define table converter that exposes more than name/creation timestamp
		TableConvertor: rest.NewDefaultTableConvertor(arc.Resource("orders")),
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return &registry.REST{Store: store}, nil
}

func NewStatusREST(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (*registry.REST, error) {
	rest, err := NewREST(scheme, optsGetter)
	if err != nil {
		return nil, err
	}

	strategy := NewStatusStrategy(scheme)
	rest.UpdateStrategy = strategy
	rest.ResetFieldsStrategy = strategy
	return rest, nil
}
