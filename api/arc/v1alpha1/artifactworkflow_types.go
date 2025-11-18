// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO
// +enum
type WorkflowPhase string

const (
	// TODO
	WorkflowPhaseInit WorkflowPhase = "Init"
)

// ArtifactWorkflowSpec specifies a single artifact which is translated into a corresponding Workflow based on its type.
type ArtifactWorkflowSpec struct {
	// Type specifies which ArtifactType is used to process this artifact.
	Type string `json:"type"`
	// The following doesn't work due to openapi-gen :/
	Parameters []ArtifactWorkflowParameter `json:"parameters,omitempty"`
	//
	SrcSecretRef corev1.LocalObjectReference `json:"srcSecretRef"`
	//
	DstSecretRef corev1.LocalObjectReference `json:"dstSecretRef"`
}

type ArtifactWorkflowParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ArtifactWorkflowStatus defines the observed state of ArtifactWorkflow
type ArtifactWorkflowStatus struct {
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
