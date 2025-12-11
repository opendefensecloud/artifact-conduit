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
var _ rest.TableConverter = &Order{}

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

func (o *Order) ConvertToTable(ctx context.Context, tableOptions runtime.Object) (*metav1.Table, error) {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Description: "Name of the Order"},
			{Name: "Created At", Type: "date", Description: "CreationTimestamp is a timestamp representing the server time when this object was created"},
			{Name: "Phase", Type: "string", Description: "Current phase of the Order"},
			{Name: "Message", Type: "string", Description: "Status message describing the current condition of the Order"},
		},
		Rows: []metav1.TableRow{
			{
				Cells: []interface{}{
					o.Name,
					o.CreationTimestamp,
					getOrderPhase(o.Status),
					o.Status.Message,
				},
				Object: runtime.RawExtension{Object: o},
			},
		},
	}
	return table, nil
}

// getOrderPhase determines the phase of an Order based on its status
func getOrderPhase(status OrderStatus) string {
	if status.Message == "" {
		return "Pending"
	}
	// Check if any artifact workflows have completed
	if len(status.ArtifactWorkflows) > 0 {
		allCompleted := true
		for _, aw := range status.ArtifactWorkflows {
			if aw.Phase != WorkflowSucceeded && aw.Phase != WorkflowFailed && aw.Phase != WorkflowError {
				allCompleted = false
				break
			}
		}
		if allCompleted {
			return "Completed"
		}
		return "Running"
	}
	return "Pending"
}
