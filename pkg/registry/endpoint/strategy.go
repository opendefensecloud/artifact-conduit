// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package endpoint

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"

	"go.opendefense.cloud/arc/api/arc"
)

// NewStrategy creates and returns a endpointStrategy instance
func NewStrategy(typer runtime.ObjectTyper) endpointStrategy {
	return endpointStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not an Endpoint
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	apiserver, ok := obj.(*arc.Endpoint)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not an Endpoint")
	}
	return labels.Set(apiserver.Labels), SelectableFields(apiserver), nil
}

// MatchEndpoint is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchEndpoint(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// SelectableFields returns a field set that represents the object.
func SelectableFields(obj *arc.Endpoint) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

type endpointStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (endpointStrategy) NamespaceScoped() bool {
	return true
}

func (endpointStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (endpointStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (endpointStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (endpointStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (endpointStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (endpointStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (endpointStrategy) Canonicalize(obj runtime.Object) {
}

func (endpointStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnUpdate returns warnings for the given update.
func (endpointStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
