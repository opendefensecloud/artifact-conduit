// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package artifacttypedefinition

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

	"github.com/opendefensecloud/artifact-conduit/api/arc"
)

// NewStrategy creates and returns a artifactTypeDefinitionStrategy instance
func NewStrategy(typer runtime.ObjectTyper) artifactTypeDefinitionStrategy {
	return artifactTypeDefinitionStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not an ArtifactTypeDefinition
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	apiserver, ok := obj.(*arc.ArtifactTypeDefinition)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not an ArtifactTypeDefinition")
	}
	return labels.Set(apiserver.Labels), SelectableFields(apiserver), nil
}

// MatchArtifactTypeDefinition is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchArtifactTypeDefinition(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// SelectableFields returns a field set that represents the object.
func SelectableFields(obj *arc.ArtifactTypeDefinition) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

type artifactTypeDefinitionStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (artifactTypeDefinitionStrategy) NamespaceScoped() bool {
	return true
}

func (artifactTypeDefinitionStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (artifactTypeDefinitionStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (artifactTypeDefinitionStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (artifactTypeDefinitionStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (artifactTypeDefinitionStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (artifactTypeDefinitionStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (artifactTypeDefinitionStrategy) Canonicalize(obj runtime.Object) {
}

func (artifactTypeDefinitionStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnUpdate returns warnings for the given update.
func (artifactTypeDefinitionStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
