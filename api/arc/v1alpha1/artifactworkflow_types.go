// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowPhase is an enum tracking in which phase a Workflow can be.
// +enum
type WorkflowPhase string

const ( // analog to Argo Workflows
	WorkflowUnknown   WorkflowPhase = ""
	WorkflowPending   WorkflowPhase = "Pending"
	WorkflowRunning   WorkflowPhase = "Running"
	WorkflowSucceeded WorkflowPhase = "Succeeded"
	WorkflowFailed    WorkflowPhase = "Failed"
	WorkflowError     WorkflowPhase = "Error"
	WorkflowActive    WorkflowPhase = "Active"
	WorkflowStopped   WorkflowPhase = "Stopped"
)

func (p WorkflowPhase) Completed() bool {
	switch p {
	case WorkflowSucceeded, WorkflowFailed, WorkflowError:
		return true
	default:
		return false
	}
}

func (p WorkflowPhase) InProgress() bool {
	switch p {
	case WorkflowPending, WorkflowRunning, WorkflowActive:
		return true
	default:
		return false
	}
}

// ArtifactWorkflowSpec specifies a single artifact which is translated into a corresponding Workflow based on its type.
type ArtifactWorkflowSpec struct {
	ArtifactWorkflowTTLSettings `json:",inline"`
	// WorkflowTemplateRef specifies the corresponding Workflow for this ArtifactWorkflow as derived from ArtifactType
	WorkflowTemplateRef ArtifactTypeTemplateRef `json:"workflowTemplateRef"`
	// Parameters defines the key-value pairs, that are passed to the underlying Workflow.
	Parameters []ArtifactWorkflowParameter `json:"parameters,omitempty"`
	// SrcSecretRef references the secret containing credentials for the source.
	SrcSecretRef corev1.LocalObjectReference `json:"srcSecretRef"`
	// DstSecretRef references the secret containing credentials for the destination.
	DstSecretRef corev1.LocalObjectReference `json:"dstSecretRef"`
	// Cron specifies options which determine when the ArtifactWorkflow should be scheduled.
	// +optional
	Cron *Cron `json:"cron,omitempty"`
}

// ArtifactWorkflowParameter represents a single key-value parameter pair.
type ArtifactWorkflowParameter struct {
	// Name is the key of the parameter.
	Name string `json:"name"`
	// Value is the string value of the parameter.
	Value string `json:"value"`
}

type WorkflowStatus struct {
	// Phase tracks which phase the corresponding Workflow is in
	Phase WorkflowPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=WorkflowPhase"`
	// A human readable message describing the current condition of the artifact workflow.
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
	// CompletionTime is the time when the workflow finished
	CompletionTime metav1.Time `json:"completionTime,omitempty"`
	// FailureTime is the time when the workflow finished with a failure status.
	FailureTime metav1.Time `json:"failureTime,omitempty"`
	// LastScheduled is the last time the workflow was scheduled via cron
	LastScheduled *metav1.Time `json:"lastScheduled,omitempty"`
	// Succeeded counts how many times child workflows succeeded
	// +optional
	Succeeded int64 `json:"succeeded" protobuf:"varint,4,rep,name=succeeded"`
	// Failed counts how many times child workflows failed
	// +optional
	Failed int64 `json:"failed" protobuf:"varint,5,rep,name=failed"`
}

// ArtifactWorkflowStatus defines the observed state of ArtifactWorkflow
type ArtifactWorkflowStatus struct {
	WorkflowStatus `json:",inline"`
	// LastReconcileAt is the last time the ArtifactWorkflow was reconciled
	LastReconcileAt metav1.Time `json:"lastReconcileAt,omitempty"`
	// LastForceAt is the last time a force reconciliation was requested
	LastForceAt metav1.Time `json:"lastForceAt,omitempty"`
	// ActiveWorkflowRef tracks the currently spawned workflow, if cron is used.
	// It resets after a successful or failed run.
	ActiveWorkflowRef corev1.LocalObjectReference `json:"activeWorkflowRef"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArtifactWorkflow is the Schema for the artifact workflows API
type ArtifactWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ArtifactWorkflowSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status ArtifactWorkflowStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArtifactWorkflowList is a list of ArtifactWorkflow objects.
type ArtifactWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []ArtifactWorkflow `json:"items" protobuf:"bytes,2,rep,name=items"`
}
