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

func (o *Endpoint) IntoTableRow() metav1.TableRow {
	return metav1.TableRow{
		Cells: []any{
			o.Name,
			o.CreationTimestamp,
			o.Spec.RemoteURL,
			o.Spec.Usage,
			o.Spec.SecretRef.Name,
		},
		Object: runtime.RawExtension{Object: o},
	}
}

func (o *Endpoint) ConvertToTable(ctx context.Context, tableOptions runtime.Object) (*metav1.Table, error) {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Description: "Name of the ArtifactType"},
			{Name: "Created At", Type: "date", Description: "CreationTimestamp is a timestamp representing the server time when this object was created"},
			{Name: "Remote URL", Type: "string", Description: "Remote location for the Endpoint"},
			{Name: "Usage", Type: "string", Description: "Usage of the Endpoint"},
			{Name: "Secret", Type: "string", Description: "Name of the Secret of the Endpoint"},
		},
		Rows: []metav1.TableRow{
			o.IntoTableRow(),
		},
	}
	table.ResourceVersion = o.GetResourceVersion()

	return table, nil
}
