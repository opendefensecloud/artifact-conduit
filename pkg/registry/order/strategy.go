// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package order

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
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"

	"go.opendefense.cloud/arc/api/arc"
)

// NewStrategy creates and returns a orderStrategy instance
func NewStrategy(typer runtime.ObjectTyper) orderStrategy {
	return orderStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not an Order
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	order, ok := obj.(*arc.Order)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not an Order")
	}
	return labels.Set(order.Labels), SelectableFields(order), nil
}

// MatchOrder is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchOrder(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// SelectableFields returns a field set that represents the object.
func SelectableFields(obj *arc.Order) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

type orderStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (orderStrategy) NamespaceScoped() bool {
	return true
}

func (orderStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (orderStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (orderStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (orderStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string { return nil }

func (orderStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (orderStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (orderStrategy) Canonicalize(obj runtime.Object) {
}

func (orderStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnUpdate returns warnings for the given update.
func (orderStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// NewStrategy creates and returns a orderStatusStrategy instance
func NewStatusStrategy(typer runtime.ObjectTyper) orderStatusStrategy {
	return orderStatusStrategy{orderStrategy{typer, names.SimpleNameGenerator}}
}

type orderStatusStrategy struct {
	orderStrategy
}

func (orderStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"arc.bwi.de/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (orderStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newOrder := obj.(*arc.Order)
	oldOrder := old.(*arc.Order)
	newOrder.Spec = oldOrder.Spec
}

func (orderStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (orderStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
