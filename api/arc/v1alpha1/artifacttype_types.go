// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArtifactTypeRules is a set of rules to be used for this type of artifact.
type ArtifactTypeRules struct {
	// SrcTypes is a list of Endpoint types, that are supported as source.
	SrcTypes []string `json:"srcTypes,omitempty"`
	// DstTypes is a list of Endpoint types, that are supported as destination.
	DstTypes []string `json:"dstType,omitempty"`
}

// ArtifactTypeSpec specifies a type of artifact and describes the corresponding workflow.
type ArtifactTypeSpec struct {
	// Rules defines a set of rules for this type.
	Rules ArtifactTypeRules `json:"rules"`
	// Parameters defines extra parameters for the Workflow to use.
	// These parameters will override parameters coming from ArtifactWorkflows.
	Parameters []ArtifactWorkflowParameter `json:"parameters"`
	// WorkflowTemplateRef specifies the corresponding Workflow for this type of artifact.
	WorkflowTemplateRef corev1.LocalObjectReference `json:"workflowTemplateRef"`
}

// ArtifactTypeStatus defines the observed state of ArtifactType
type ArtifactTypeStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArtifactType is the Schema for the endpoints API
type ArtifactType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ArtifactTypeSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status ArtifactTypeStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArtifactTypeList is a list of ArtifactType objects.
type ArtifactTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []ArtifactType `json:"items" protobuf:"bytes,2,rep,name=items"`
}
