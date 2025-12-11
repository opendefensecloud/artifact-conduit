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

var _ resource.Object = &Endpoint{}
var _ rest.Validater = &Endpoint{}
var _ rest.ValidateUpdater = &Endpoint{}

func (o *Endpoint) GetObjectMeta() *metav1.ObjectMeta {
	return &o.ObjectMeta
}

func (o *Endpoint) NamespaceScoped() bool {
	return true
}

func (o *Endpoint) New() runtime.Object {
	return &Endpoint{}
}

func (o *Endpoint) NewList() runtime.Object {
	return &EndpointList{}
}

func (o *Endpoint) GetGroupResource() schema.GroupResource {
	return SchemeGroupVersion.WithResource("endpoints").GroupResource()
}

func (o *Endpoint) Validate(ctx context.Context) field.ErrorList {
	return validateEndpoint(o)
}

func (o *Endpoint) ValidateUpdate(ctx context.Context, old runtime.Object) field.ErrorList {
	return validateEndpoint(o)
}

func validateEndpoint(o *Endpoint) field.ErrorList {
	allErrs := field.ErrorList{}

	if o.Spec.RemoteURL == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "remoteURL"), "remoteURL is required"))
	}

	return allErrs
}
