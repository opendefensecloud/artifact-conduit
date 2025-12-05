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

var _ resource.Object = &ArtifactWorkflow{}
var _ resource.ObjectWithStatusSubResource = &ArtifactWorkflow{}
var _ rest.Validater = &ArtifactWorkflow{}
var _ rest.ValidateUpdater = &ArtifactWorkflow{}

func (o *ArtifactWorkflow) GetObjectMeta() *metav1.ObjectMeta {
	return &o.ObjectMeta
}

func (o *ArtifactWorkflow) NamespaceScoped() bool {
	return true
}

func (o *ArtifactWorkflow) New() runtime.Object {
	return &ArtifactWorkflow{}
}

func (o *ArtifactWorkflow) NewList() runtime.Object {
	return &ArtifactWorkflowList{}
}

func (o *ArtifactWorkflow) GetGroupResource() schema.GroupResource {
	return SchemeGroupVersion.WithResource("artifactworkflows").GroupResource()
}

func (o *ArtifactWorkflow) CopyStatusTo(obj runtime.Object) {
	if obj, ok := obj.(*ArtifactWorkflow); ok {
		obj.Status = o.Status
	}
}

func (o *ArtifactWorkflow) Validate(ctx context.Context) field.ErrorList {
	allErrs := field.ErrorList{}
	paramPath := field.NewPath("spec", "parameters")

	// Check for duplicate parameter names in ArtifactWorkflow
	seen := map[string]int{}
	for i, param := range o.Spec.Parameters {
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

func (o *ArtifactWorkflow) ValidateUpdate(ctx context.Context, old runtime.Object) field.ErrorList {
	oldArtifactWorkflow, ok := old.(*ArtifactWorkflow)
	if !ok {
		return field.ErrorList{field.InternalError(field.NewPath(""), fmt.Errorf("old object is not an ArtifactWorkflow"))}
	}

	allErrs := field.ErrorList{}

	// Check if spec has been modified (spec should be immutable)
	if !reflect.DeepEqual(o.Spec, oldArtifactWorkflow.Spec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "spec is immutable and cannot be updated"))
	}

	return allErrs
}
