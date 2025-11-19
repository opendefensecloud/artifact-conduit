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
	case WorkflowPending, WorkflowRunning:
		return true
	default:
		return false
	}
}

// ArtifactWorkflowSpec specifies a single artifact which is translated into a corresponding Workflow based on its type.
type ArtifactWorkflowSpec struct {
	// Type specifies which ArtifactType is used to process this artifact.
	Type string `json:"type"`
	// Parameters defines the key-value pairs, that are passed to the underlying Workflow.
	Parameters []ArtifactWorkflowParameter `json:"parameters,omitempty"`
	// SrcSecretRef references the secret containing credentials for the source.
	SrcSecretRef corev1.LocalObjectReference `json:"srcSecretRef"`
	// DstSecretRef references the secret containing credentials for the destination.
	DstSecretRef corev1.LocalObjectReference `json:"dstSecretRef"`
}

// ArtifactWorkflowParameter represents a single key-value parameter pair.
type ArtifactWorkflowParameter struct {
	// Name is the key of the parameter.
	Name string `json:"name"`
	// Value is the string value of the parameter.
	Value string `json:"value"`
}

// ArtifactWorkflowStatus defines the observed state of ArtifactWorkflow
type ArtifactWorkflowStatus struct {
	// Phase tracks which phase the corresponding Workflow is in
	Phase WorkflowPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=WorkflowPhase"`
	// A human readable message describing the current condition of the artifact workflow.
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArtifactWorkflow is the Schema for the fragments API
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
