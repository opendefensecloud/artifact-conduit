// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc

import (
	"context"

	"go.opendefense.cloud/kit/apiserver/resource"
	"go.opendefense.cloud/kit/apiserver/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ resource.Object = &Order{}
var _ resource.ObjectWithStatusSubResource = &Order{}
var _ rest.Validater = &Order{}
var _ rest.ValidateUpdater = &Order{}

func (o *Order) GetObjectMeta() *metav1.ObjectMeta {
	return &o.ObjectMeta
}

func (o *Order) NamespaceScoped() bool {
	return true
}

func (o *Order) New() runtime.Object {
	return &Order{}
}

func (o *Order) NewList() runtime.Object {
	return &OrderList{}
}

func (o *Order) GetGroupResource() schema.GroupResource {
	return SchemeGroupVersion.WithResource("orders").GroupResource()
}

func (o *Order) CopyStatusTo(obj runtime.Object) {
	if obj, ok := obj.(*Order); ok {
		obj.Status = o.Status
	}
}

func (o *Order) Validate(ctx context.Context) field.ErrorList {
	return validateOrder(o)
}

func (o *Order) ValidateUpdate(ctx context.Context, old runtime.Object) field.ErrorList {
	return validateOrder(o)
}

func validateOrder(o *Order) field.ErrorList {
	allErrs := field.ErrorList{}

	hasDefaultSrc := o.Spec.Defaults.SrcRef.Name != ""
	hasDefaultDst := o.Spec.Defaults.DstRef.Name != ""
	for i, artifact := range o.Spec.Artifacts {
		if artifact.Type == "" {
			allErrs = append(allErrs, field.Required(field.NewPath("spec", "artifacts").Index(i).Child("type"), "type is required"))
		}
		if artifact.SrcRef.Name == "" && !hasDefaultSrc {
			allErrs = append(allErrs, field.Required(field.NewPath("spec", "artifacts").Index(i).Child("srcRef"), "source endpoint has to be specified without default source"))
		}
		if artifact.DstRef.Name == "" && !hasDefaultDst {
			allErrs = append(allErrs, field.Required(field.NewPath("spec", "artifacts").Index(i).Child("dstRef"), "destination endpoint has to be specified without default destination"))
		}
	}

	return allErrs
}
