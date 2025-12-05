// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage/names"
)

type Scoper rest.Scoper

type NameGenerator names.NameGenerator

// AllowCreateOnUpdater implements a subset of rest.RESTUpdateStrategy and
// it can be used by objects to override DefaultStrategy behaviour.
type AllowCreateOnUpdater interface {
	// AllowCreateOnUpdate returns true if the object can be created by a PUT.
	AllowCreateOnUpdate() bool
}

// AllowUnconditionalUpdater implements a subset of rest.RESTUpdateStrategy and
// it can be used by objects to override DefaultStrategy behaviour.
type AllowUnconditionalUpdater interface {
	// AllowUnconditionalUpdate returns true if the object can be updated
	// unconditionally (irrespective of the latest resource version), when
	// there is no resource version specified in the object.
	AllowUnconditionalUpdate() bool
}

// Canonicalizer implements a subset of rest.RESTUpdateStrategy/rest.RESTCreateStrategy and
// it can be used by objects to override DefaultStrategy behaviour.
type Canonicalizer interface {
	// Canonicalize allows an object to be mutated into a canonical form. This
	// ensures that code that operates on these objects can rely on the common
	// form for things like comparison.  Canonicalize is invoked after
	// validation has succeeded but before the object has been persisted.
	// This method may mutate the object.
	Canonicalize()
}

// PrepareForCreater implements a subset of rest.RESTCreateStrategy and
// it can be used by objects to override DefaultStrategy behaviour.
type PrepareForCreater interface {
	// PrepareForCreate is invoked on create before validation to normalize
	// the object.  For example: remove fields that are not to be persisted,
	// sort order-insensitive list fields, etc.  This should not remove fields
	// whose presence would be considered a validation error.
	//
	// Often implemented as a type check and an initailization or clearing of
	// status. Clear the status because status changes are internal. External
	// callers of an api (users) should not be setting an initial status on
	// newly created objects.
	PrepareForCreate(ctx context.Context)
}

// PrepareForUpdater implements a subset of rest.RESTUpdateStrategy and
// it can be used by objects to override DefaultStrategy behaviour.
type PrepareForUpdater interface {
	// PrepareForUpdate is invoked on update before validation to normalize
	// the object.  For example: remove fields that are not to be persisted,
	// sort order-insensitive list fields, etc.  This should not remove fields
	// whose presence would be considered a validation error.
	PrepareForUpdate(ctx context.Context, old runtime.Object)
}

// TableConverter implements an adapted version of rest.TableConverter
// it can be used by objects to override DefaultStrategy behaviour.
type TableConverter interface {
	ConvertToTable(ctx context.Context, tableOptions runtime.Object) (*metav1.Table, error)
}

// Validater implements a subset of rest.RESTCreateStrategy and
// it can be used by objects to override DefaultStrategy behaviour.
type Validater interface {
	// Validate returns an ErrorList with validation errors or nil.  Validate
	// is invoked after default fields in the object have been filled in
	// before the object is persisted.  This method should not mutate the
	// object.
	Validate(ctx context.Context) field.ErrorList
}

// Validater implements a subset of rest.RESTUpdateStrategy and
// it can be used by objects to override DefaultStrategy behaviour.
type ValidateUpdater interface {
	// ValidateUpdate is invoked after default fields in the object have been
	// filled in before the object is persisted.  This method should not mutate
	// the object.
	ValidateUpdate(ctx context.Context, obj runtime.Object) field.ErrorList
}
