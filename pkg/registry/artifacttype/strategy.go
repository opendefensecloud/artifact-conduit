// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package artifacttype

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

	"go.opendefense.cloud/arc/api/arc"
)

// NewStrategy creates and returns a artifactTypeDefinitionStrategy instance
func NewStrategy(typer runtime.ObjectTyper) artifactTypeStrategy {
	return artifactTypeStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not an ArtifactType
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	apiserver, ok := obj.(*arc.ArtifactType)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not an ArtifactType")
	}
	return labels.Set(apiserver.Labels), SelectableFields(apiserver), nil
}

// MatchArtifactType is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchArtifactType(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// SelectableFields returns a field set that represents the object.
func SelectableFields(obj *arc.ArtifactType) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

type artifactTypeStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (artifactTypeStrategy) NamespaceScoped() bool {
	return true
}

func (artifactTypeStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (artifactTypeStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (artifactTypeStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	artifactType, ok := obj.(*arc.ArtifactType)
	if !ok {
		return field.ErrorList{field.Invalid(field.NewPath(""), obj, "expected ArtifactType object")}
	}

	allErrs := field.ErrorList{}

	// Validate Parameters - check for empty names and duplicates
	paramNames := make(map[string]bool)
	for i, param := range artifactType.Spec.Parameters {
		switch {
		case param.Name == "":
			allErrs = append(allErrs, field.Required(field.NewPath("spec", "parameters").Index(i).Child("name"), "parameter name is required"))
		case paramNames[param.Name]:
			allErrs = append(allErrs, field.Duplicate(field.NewPath("spec", "parameters").Index(i).Child("name"), param.Name))
		default:
			paramNames[param.Name] = true
		}
	}

	return allErrs
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (artifactTypeStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (artifactTypeStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (artifactTypeStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (artifactTypeStrategy) Canonicalize(obj runtime.Object) {
}

func (artifactTypeStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newArtifactType, ok := obj.(*arc.ArtifactType)
	if !ok {
		return field.ErrorList{field.InternalError(field.NewPath(""), fmt.Errorf("not an ArtifactType"))}
	}

	oldArtifactType, ok := old.(*arc.ArtifactType)
	if !ok {
		return field.ErrorList{field.InternalError(field.NewPath(""), fmt.Errorf("old object is not an ArtifactType"))}
	}

	allErrs := field.ErrorList{}

	// Check if spec has been modified (spec should be immutable)
	if !reflect.DeepEqual(newArtifactType.Spec, oldArtifactType.Spec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "spec is immutable and cannot be updated"))
	}

	return allErrs
}

// WarningsOnUpdate returns warnings for the given update.
func (artifactTypeStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
