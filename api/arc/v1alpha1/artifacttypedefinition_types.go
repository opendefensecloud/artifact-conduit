// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArtifactTypeDefinitionRules is a set of rules to be used for this type of artifact.
type ArtifactTypeDefinitionRules struct {
	// SrcTypes is a list of Endpoint types, that are supported as source.
	SrcTypes []string `json:"srcTypes,omitempty"`
	// DstTypes is a list of Endpoint types, that are supported as destination.
	DstTypes []string `json:"dstType,omitempty"`
}

// ArtifactTypeDefinitionSpec specifies a type of artifact and describes the corresponding workflow.
type ArtifactTypeDefinitionSpec struct {
	// Rules defines a set of rules for this type.
	Rules ArtifactTypeDefinitionRules `json:"rules"`
	// Defaults optionally sets defaults for this type of artifact.
	// +optional
	Defaults OrderDefaults `json:"defaults,omitempty"`
	// WorkflowTemplateRef specifies the corresponding Workflow for this type of artifact.
	WorkflowTemplateRef corev1.LocalObjectReference `json:"workflowTemplateRef"`
}

// ArtifactTypeDefinitionStatus defines the observed state of ArtifactTypeDefinition
type ArtifactTypeDefinitionStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArtifactTypeDefinition is the Schema for the endpoints API
type ArtifactTypeDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ArtifactTypeDefinitionSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status ArtifactTypeDefinitionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArtifactTypeDefinitionList is a list of ArtifactTypeDefinition objects.
type ArtifactTypeDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []ArtifactTypeDefinition `json:"items" protobuf:"bytes,2,rep,name=items"`
}
