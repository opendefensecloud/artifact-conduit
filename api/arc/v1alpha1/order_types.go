// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OrderDefaults is used to set defaults for all other artifacts of an Order.
type OrderDefaults struct {
	// SrcRef defines which Endpoint object is used as fallback source by all artifacts.
	// +optional
	SrcRef corev1.LocalObjectReference `json:"srcRef,omitempty"`
	// DstRef defines which Endpoint object is used as fallback destination by all artifacts.
	// +optional
	DstRef corev1.LocalObjectReference `json:"dstRef,omitempty"`
}

// OrderArtifact specifies a single artifact which is translated into a corresponding OrderArtifactWorkflow
type OrderArtifact struct {
	// Type specifies which ArtifactType is used to process this artifact.
	Type string `json:"type"`
	// SrcRef defines which Endpoint object is used as source (falls back to OrderDefaults).
	// +optional
	SrcRef corev1.LocalObjectReference `json:"srcRef,omitempty"`
	// SrcRef defines which Endpoint object is used as destination (falls back to OrderDefaults).
	// +optional
	DstRef corev1.LocalObjectReference `json:"dstRef,omitempty"`
	// Spec specifies parameters used by the underlying Workflow.
	Spec runtime.RawExtension `json:"spec,omitempty"`
}

// OrderSpec defines the desired state of Order
type OrderSpec struct {
	// Defaults sets up defaults for all artifacts.
	// +optional
	Defaults OrderDefaults `json:"defaults,omitempty"`
	// Artifacts lists all artifacts, that will be processed by this Order.
	Artifacts []OrderArtifact `json:"artifacts,omitempty"`
}

// OrderStatus defines the observed state of Order
type OrderStatus struct {
	// ArtifactWorkflows tracks the created workflows
	ArtifactWorkflows map[string]OrderArtifactWorkflowStatus `json:"artifactWorkflows,omitempty"`
}

type OrderArtifactWorkflowStatus struct {
	// ArtifactIndex references back the index the corresponding artifact has in the .Spec
	ArtifactIndex int `json:"artifactIndex"`
	// Phase tracks which phase the corresponding Workflow is in
	Phase WorkflowPhase `json:"phase"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Order is the Schema for the orders API
type Order struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   OrderSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status OrderStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OrderList is a list of Order objects.
type OrderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Order `json:"items" protobuf:"bytes,2,rep,name=items"`
}
