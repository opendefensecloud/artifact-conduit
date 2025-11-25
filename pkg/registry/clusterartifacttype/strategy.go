// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package clusterartifacttype

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

// NewStrategy creates and returns a artifactTypeDefinitionStrategy instance
func NewStrategy(typer runtime.ObjectTyper) clusterArtifactTypeStrategy {
	return clusterArtifactTypeStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not an ArtifactType
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	apiserver, ok := obj.(*arc.ClusterArtifactType)
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
func SelectableFields(obj *arc.ClusterArtifactType) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

type clusterArtifactTypeStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (clusterArtifactTypeStrategy) NamespaceScoped() bool {
	return false
}

func (clusterArtifactTypeStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (clusterArtifactTypeStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (clusterArtifactTypeStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (clusterArtifactTypeStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (clusterArtifactTypeStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (clusterArtifactTypeStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (clusterArtifactTypeStrategy) Canonicalize(obj runtime.Object) {
}

func (clusterArtifactTypeStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnUpdate returns warnings for the given update.
func (clusterArtifactTypeStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
