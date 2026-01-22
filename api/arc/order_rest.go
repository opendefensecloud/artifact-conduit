// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc

import (
	"context"
	"fmt"
	"time"

	rcron "github.com/robfig/cron/v3"
	"go.opendefense.cloud/kit/apiserver/resource"
	"go.opendefense.cloud/kit/apiserver/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const DefaultCronMinScheduleInterval = 10 * time.Minute

var CronMinScheduleInterval = DefaultCronMinScheduleInterval

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
		if artifact.Cron != nil {
			if err := validateCron(field.NewPath("spec", "artifacts").Index(i).Child("cron"), artifact.Cron); err != nil {
				allErrs = append(allErrs, err)
			}
		}
	}

	if o.Spec.Defaults.Cron != nil {
		if err := validateCron(field.NewPath("spec", "defaults").Child("cron"), o.Spec.Defaults.Cron); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	return allErrs
}

func validateCron(path *field.Path, cron *Cron) *field.Error {
	if len(cron.Schedules) == 0 {
		return field.Required(path.Child("schedules"), "if cron is specified, schedules can not be empty")
	}

	for i, expr := range cron.Schedules {
		schedule, err := rcron.ParseStandard(expr)
		if err != nil {
			return field.Invalid(path.Child("schedules").Index(i), expr, "schedule is not an valid cron expression or supported descriptor")
		}

		t1 := schedule.Next(time.Now())
		t2 := schedule.Next(t1)

		interval := t2.Sub(t1)
		if interval < CronMinScheduleInterval {
			return field.Invalid(path.Child("schedules").Index(i), expr, fmt.Sprintf("schedule has an interval smaller than configured constraints. Expected value to have larger interval than %s", CronMinScheduleInterval))
		}
	}
	return nil
}

func (o *Order) IntoTableRow() metav1.TableRow {
	return metav1.TableRow{
		Cells: []any{
			o.Name,
			getOrderPhase(o.Status),
			duration.HumanDuration(metav1.Now().Sub(o.CreationTimestamp.Time)),
		},
		Object: runtime.RawExtension{Object: o},
	}
}

func (o *Order) ConvertToTable(ctx context.Context, tableOptions runtime.Object) (*metav1.Table, error) {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Description: "Name of the Order"},
			{Name: "Phase", Type: "string", Description: "Current phase of the Order"},
			{Name: "Age", Type: "string", Description: "Time since creation of the Order"},
		},
		Rows: []metav1.TableRow{
			o.IntoTableRow(),
		},
	}
	table.ResourceVersion = o.GetResourceVersion()
	return table, nil
}

// getOrderPhase determines the phase of an Order based on its status
func getOrderPhase(status OrderStatus) string {
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
