// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package artifactworkflow

import (
	"context"
	"fmt"
	"reflect"

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

// NewStrategy creates and returns a artifactWorkflowStrategy instance
func NewStrategy(typer runtime.ObjectTyper) artifactWorkflowStrategy {
	return artifactWorkflowStrategy{typer, names.SimpleNameGenerator}
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

type artifactWorkflowStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (artifactWorkflowStrategy) NamespaceScoped() bool {
	return true
}

func (artifactWorkflowStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (artifactWorkflowStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (artifactWorkflowStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	artifactWorkflow, ok := obj.(*arc.ArtifactWorkflow)
	if !ok {
		// TOT: Check if errors are logged automatically.
		return field.ErrorList{field.InternalError(field.NewPath(""), fmt.Errorf("not an ArtifactWorkflow"))}
	}

	allErrs := field.ErrorList{}
	paramPath := field.NewPath("spec", "parameters")

	// Check for duplicate parameter names in ArtifactWorkflow
	seen := map[string]int{}
	for i, param := range artifactWorkflow.Spec.Parameters {
		if idx, exists := seen[param.Name]; exists {
			// Only add error for the first occurrence once
			if idx >= 0 {
				allErrs = append(allErrs, field.Duplicate(paramPath.Index(idx).Child("name"), param.Name))
				seen[param.Name] = -1 // Mark as already reported
			}
			allErrs = append(allErrs, field.Duplicate(paramPath.Index(i).Child("name"), param.Name))
		} else {
			seen[param.Name] = i
		}
	}
	return allErrs
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (artifactWorkflowStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (artifactWorkflowStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (artifactWorkflowStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (artifactWorkflowStrategy) Canonicalize(obj runtime.Object) {
}

func (artifactWorkflowStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newArtifactWorkflow, ok := obj.(*arc.ArtifactWorkflow)
	if !ok {
		return field.ErrorList{field.InternalError(field.NewPath(""), fmt.Errorf("not an ArtifactWorkflow"))}
	}

	oldArtifactWorkflow, ok := old.(*arc.ArtifactWorkflow)
	if !ok {
		return field.ErrorList{field.InternalError(field.NewPath(""), fmt.Errorf("old object is not an ArtifactWorkflow"))}
	}

	allErrs := field.ErrorList{}

	// Check if spec has been modified (spec should be immutable)
	if !reflect.DeepEqual(newArtifactWorkflow.Spec, oldArtifactWorkflow.Spec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "spec is immutable and cannot be updated"))
	}

	return allErrs
}

// WarningsOnUpdate returns warnings for the given update.
func (artifactWorkflowStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// NewStatusStrategy creates and returns a artifactWorkflowStatusStrategy instance
func NewStatusStrategy(typer runtime.ObjectTyper) artifactWorkflowStatusStrategy {
	return artifactWorkflowStatusStrategy{artifactWorkflowStrategy{typer, names.SimpleNameGenerator}}
}

type artifactWorkflowStatusStrategy struct {
	artifactWorkflowStrategy
}

func (artifactWorkflowStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"arc.bwi.de/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (artifactWorkflowStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newArtifactWorkflow := obj.(*arc.ArtifactWorkflow)
	oldArtifactWorkflow := old.(*arc.ArtifactWorkflow)
	newArtifactWorkflow.Spec = oldArtifactWorkflow.Spec
}

func (artifactWorkflowStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (artifactWorkflowStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
