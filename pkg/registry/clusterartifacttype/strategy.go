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
	// Type assertion
	clusterArtifactType, ok := obj.(*arc.ClusterArtifactType)
	if !ok {
		return field.ErrorList{field.Invalid(field.NewPath(""), obj, "expected ClusterArtifactType")}
	}
	// Individual validations
	allErrs := field.ErrorList{}
	specPath := field.NewPath("spec")
	// rulesPath := specPath.Child("rules")
	// // Validate SrcTypes and DstTypes
	// for i, srcType := range clusterArtifactType.Spec.Rules.SrcTypes {
	// 	if srcType == "" {
	// 		allErrs = append(allErrs, field.Required(rulesPath.Child("srcTypes").Index(i), "source type cannot be empty"))
	// 	}
	// }
	// for i, dstType := range clusterArtifactType.Spec.Rules.DstTypes {
	// 	if dstType == "" {
	// 		allErrs = append(allErrs, field.Required(rulesPath.Child("dstTypes").Index(i), "destination type cannot be empty"))
	// 	}
	// }
	// Validate parameters - only check for duplicate values
	parametersPath := specPath.Child("parameters")
	seenParams := make(map[string]bool)
	for i, param := range clusterArtifactType.Spec.Parameters {
		paramPath := parametersPath.Index(i)
		// switch {
		// case param.Name == "":
		// 	allErrs = append(allErrs, field.Required(paramPath.Child("name"), "parameter name cannot be empty"))
		// case seenParams[param.Name]:
		if seenParams[param.Name] {
			allErrs = append(allErrs, field.Duplicate(paramPath.Child("name"), param.Name))
		} else {
			seenParams[param.Name] = true
		}
		// default:
		// 	seenParams[param.Name] = true
		// }
	}
	// // Validate WorkflowTemplateRef
	// templateRefPath := specPath.Child("workflowTemplateRef")
	// if clusterArtifactType.Spec.WorkflowTemplateRef.Name == "" {
	// 	allErrs = append(allErrs, field.Required(templateRefPath.Child("name"), "workflow template reference name is required"))
	// }
	// // Validate ClusterScope to be true for ClusterArtifactType
	// if !clusterArtifactType.Spec.WorkflowTemplateRef.ClusterScope {
	// 	allErrs = append(allErrs, field.Invalid(templateRefPath.Child("clusterScope"),
	// 		clusterArtifactType.Spec.WorkflowTemplateRef.ClusterScope,
	// 		"ClusterArtifactType must reference cluster-scoped workflow templates"))
	// }
	return allErrs
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
	// Type assertion
	clusterArtifactType, ok := obj.(*arc.ClusterArtifactType)
	if !ok {
		return field.ErrorList{field.Invalid(field.NewPath(""), obj, "expected ClusterArtifactType")}
	}
	// Individual validations
	allErrs := field.ErrorList{}
	specPath := field.NewPath("spec")
	rulesPath := specPath.Child("rules")
	// Validate SrcTypes and DstTypes
	for i, srcType := range clusterArtifactType.Spec.Rules.SrcTypes {
		if srcType == "" {
			allErrs = append(allErrs, field.Required(rulesPath.Child("srcTypes").Index(i), "source type cannot be empty"))
		}
	}
	for i, dstType := range clusterArtifactType.Spec.Rules.DstTypes {
		if dstType == "" {
			allErrs = append(allErrs, field.Required(rulesPath.Child("dstTypes").Index(i), "destination type cannot be empty"))
		}
	}
	// Validate parameters
	parametersPath := specPath.Child("parameters")
	seenParams := make(map[string]bool)
	for i, param := range clusterArtifactType.Spec.Parameters {
		paramPath := parametersPath.Index(i)
		switch {
		case param.Name == "":
			allErrs = append(allErrs, field.Required(paramPath.Child("name"), "parameter name cannot be empty"))
		case seenParams[param.Name]:
			allErrs = append(allErrs, field.Duplicate(paramPath.Child("name"), param.Name))
		default:
			seenParams[param.Name] = true
		}
	}
	// Validate WorkflowTemplateRef
	templateRefPath := specPath.Child("workflowTemplateRef")
	if clusterArtifactType.Spec.WorkflowTemplateRef.Name == "" {
		allErrs = append(allErrs, field.Required(templateRefPath.Child("name"), "workflow template reference name is required"))
	}
	// Validate ClusterScope to be true for ClusterArtifactType
	if !clusterArtifactType.Spec.WorkflowTemplateRef.ClusterScope {
		allErrs = append(allErrs, field.Invalid(templateRefPath.Child("clusterScope"),
			clusterArtifactType.Spec.WorkflowTemplateRef.ClusterScope,
			"ClusterArtifactType must reference cluster-scoped workflow templates"))
	}
	return allErrs
}

// WarningsOnUpdate returns warnings for the given update.
func (clusterArtifactTypeStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
