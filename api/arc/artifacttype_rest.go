// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc

import (
	"context"
	"fmt"
	"reflect"

	"go.opendefense.cloud/kit/apiserver/resource"
	"go.opendefense.cloud/kit/apiserver/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ resource.Object = &ArtifactType{}
var _ rest.Validater = &ArtifactType{}
var _ rest.ValidateUpdater = &ArtifactType{}

func (o *ArtifactType) GetObjectMeta() *metav1.ObjectMeta {
	return &o.ObjectMeta
}

func (o *ArtifactType) NamespaceScoped() bool {
	return true
}

func (o *ArtifactType) New() runtime.Object {
	return &ArtifactType{}
}

func (o *ArtifactType) NewList() runtime.Object {
	return &ArtifactTypeList{}
}

func (o *ArtifactType) GetGroupResource() schema.GroupResource {
	return SchemeGroupVersion.WithResource("artifacttypes").GroupResource()
}

func (o *ArtifactType) Validate(ctx context.Context) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate Parameters - check for empty names and duplicates
	paramNames := make(map[string]bool)
	for i, param := range o.Spec.Parameters {
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

func (o *ArtifactType) ValidateUpdate(ctx context.Context, old runtime.Object) field.ErrorList {
	oldArtifactType, ok := old.(*ArtifactType)
	if !ok {
		return field.ErrorList{field.InternalError(field.NewPath(""), fmt.Errorf("old object is not an ArtifactType"))}
	}

	allErrs := field.ErrorList{}

	// Check if spec has been modified (spec should be immutable)
	if !reflect.DeepEqual(o.Spec, oldArtifactType.Spec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "spec is immutable and cannot be updated"))
	}

	return allErrs
}

var _ resource.Object = &ClusterArtifactType{}
var _ rest.Validater = &ClusterArtifactType{}
var _ rest.ValidateUpdater = &ClusterArtifactType{}

func (o *ClusterArtifactType) GetObjectMeta() *metav1.ObjectMeta {
	return &o.ObjectMeta
}

func (o *ClusterArtifactType) NamespaceScoped() bool {
	return false
}

func (o *ClusterArtifactType) New() runtime.Object {
	return &ClusterArtifactType{}
}

func (o *ClusterArtifactType) NewList() runtime.Object {
	return &ClusterArtifactTypeList{}
}

func (o *ClusterArtifactType) GetGroupResource() schema.GroupResource {
	return SchemeGroupVersion.WithResource("clusterartifacttypes").GroupResource()
}

func (o *ClusterArtifactType) Validate(ctx context.Context) field.ErrorList {
	allErrs := field.ErrorList{}
	specPath := field.NewPath("spec")

	// Validate parameters - only check for duplicate values
	parametersPath := specPath.Child("parameters")
	seenParams := make(map[string]bool)
	for i, param := range o.Spec.Parameters {
		paramPath := parametersPath.Index(i)
		if seenParams[param.Name] {
			allErrs = append(allErrs, field.Duplicate(paramPath.Child("name"), param.Name))
		} else {
			seenParams[param.Name] = true
		}
	}
	return allErrs
}

func (o *ClusterArtifactType) ValidateUpdate(ctx context.Context, old runtime.Object) field.ErrorList {
	allErrs := field.ErrorList{}
	specPath := field.NewPath("spec")
	rulesPath := specPath.Child("rules")

	// Validate SrcTypes and DstTypes
	for i, srcType := range o.Spec.Rules.SrcTypes {
		if srcType == "" {
			allErrs = append(allErrs, field.Required(rulesPath.Child("srcTypes").Index(i), "source type cannot be empty"))
		}
	}
	for i, dstType := range o.Spec.Rules.DstTypes {
		if dstType == "" {
			allErrs = append(allErrs, field.Required(rulesPath.Child("dstTypes").Index(i), "destination type cannot be empty"))
		}
	}
	// Validate parameters
	parametersPath := specPath.Child("parameters")
	seenParams := make(map[string]bool)
	for i, param := range o.Spec.Parameters {
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
	if o.Spec.WorkflowTemplateRef.Name == "" {
		allErrs = append(allErrs, field.Required(templateRefPath.Child("name"), "workflow template reference name is required"))
	}
	return allErrs
}
