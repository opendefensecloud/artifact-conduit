// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package fragment

import (
	"gitlab.opencode.de/bwi/ace/artifact-conduit/api/arc"
	"gitlab.opencode.de/bwi/ace/artifact-conduit/pkg/registry"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (*registry.REST, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc:                   func() runtime.Object { return &arc.Fragment{} },
		NewListFunc:               func() runtime.Object { return &arc.FragmentList{} },
		PredicateFunc:             MatchFragment,
		DefaultQualifiedResource:  arc.Resource("fragments"),
		SingularQualifiedResource: arc.Resource("fragment"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		// TODO: define table converter that exposes more than name/creation timestamp
		TableConvertor: rest.NewDefaultTableConvertor(arc.Resource("fragments")),
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return &registry.REST{Store: store}, nil
}
