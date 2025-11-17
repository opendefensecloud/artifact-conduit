// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package fragment

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

// NewStrategy creates and returns a fragmentStrategy instance
func NewStrategy(typer runtime.ObjectTyper) fragmentStrategy {
	return fragmentStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not an ArtifactWorkflow
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	apiserver, ok := obj.(*arc.ArtifactWorkflow)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not an ArtifactWorkflow")
	}
	return labels.Set(apiserver.Labels), SelectableFields(apiserver), nil
}

// MatchArtifactWorkflow is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchArtifactWorkflow(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// SelectableFields returns a field set that represents the object.
func SelectableFields(obj *arc.ArtifactWorkflow) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

type fragmentStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (fragmentStrategy) NamespaceScoped() bool {
	return true
}

func (fragmentStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (fragmentStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (fragmentStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (fragmentStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (fragmentStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (fragmentStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (fragmentStrategy) Canonicalize(obj runtime.Object) {
}

func (fragmentStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	// TODO: fragments should be immutable
	return field.ErrorList{}
}

// WarningsOnUpdate returns warnings for the given update.
func (fragmentStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// NewStrategy creates and returns a fragmentStatusStrategy instance
func NewStatusStrategy(typer runtime.ObjectTyper) fragmentStatusStrategy {
	return fragmentStatusStrategy{fragmentStrategy{typer, names.SimpleNameGenerator}}
}

type fragmentStatusStrategy struct {
	fragmentStrategy
}

func (fragmentStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"arc.bwi.de/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (fragmentStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newArtifactWorkflow := obj.(*arc.ArtifactWorkflow)
	oldArtifactWorkflow := old.(*arc.ArtifactWorkflow)
	newArtifactWorkflow.Spec = oldArtifactWorkflow.Spec
}

func (fragmentStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (fragmentStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
