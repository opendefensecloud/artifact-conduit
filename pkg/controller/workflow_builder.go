// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"go.opendefense.cloud/arc/api/arc/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type workflowBuilder struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	cron *v1alpha1.Cron
	spec wfv1alpha1.WorkflowSpec
}

type WorkflowBuilderOption func(b *workflowBuilder)

func NewWorkflowBuilder(opts ...WorkflowBuilderOption) *workflowBuilder {
	b := &workflowBuilder{}
	for _, o := range opts {
		o(b)
	}
	return b
}

func NewWorkflowBuilderWithName(namespace, name string, opts ...WorkflowBuilderOption) *workflowBuilder {
	b := &workflowBuilder{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	for _, o := range opts {
		o(b)
	}
	return b
}

func WithObjectMeta(meta metav1.ObjectMeta) WorkflowBuilderOption {
	return func(b *workflowBuilder) {
		b.ObjectMeta = meta
	}
}

func WithCron(cron *v1alpha1.Cron) WorkflowBuilderOption {
	return func(b *workflowBuilder) {
		b.cron = cron
	}
}

func WithWorkflowSpec(spec wfv1alpha1.WorkflowSpec) WorkflowBuilderOption {
	return func(b *workflowBuilder) {
		b.spec = spec
	}
}

func (b *workflowBuilder) Build() client.Object {
	if b.cron == nil {
		return &wfv1alpha1.Workflow{
			ObjectMeta: b.ObjectMeta,
			Spec:       b.spec,
		}
	}

	return &wfv1alpha1.CronWorkflow{
		ObjectMeta: b.ObjectMeta,
		Spec: wfv1alpha1.CronWorkflowSpec{
			WorkflowSpec:            b.spec,
			Schedules:               b.cron.Schedules,
			ConcurrencyPolicy:       wfv1alpha1.ConcurrencyPolicy(b.cron.ConcurrencyPolicy),
			StartingDeadlineSeconds: b.cron.StartingDeadlineSeconds,
			Timezone:                b.cron.Timezone,
			When:                    b.cron.When,
		},
	}
}
