// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:validation:Enum=Allow;Forbid;Replace
type ConcurrencyPolicy string

const (
	AllowConcurrent   ConcurrencyPolicy = "Allow"
	ForbidConcurrent  ConcurrencyPolicy = "Forbid"
	ReplaceConcurrent ConcurrencyPolicy = "Replace"
)

// OrderCron represents an order's cron schedule.
type OrderCron struct {
	// Timezone is the timezone against which the cron schedule will be calculated, e.g. "Asia/Tokyo". Default is machine's local time.
	Timezone string `json:"timezone,omitempty"`
	// ConcurrencyPolicy is the K8s-style concurrency policy that will be used
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`
	// StartingDeadlineSeconds is the K8s-style deadline that will limit the time a Order will be run after its
	// original scheduled time if it is missed.
	// +kubebuilder:validation:Minimum=0
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`
	// Schedules is a list of schedules to run the Order in Cron format
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:Pattern=`^(@(yearly|annually|monthly|weekly|daily|midnight|hourly)|@every\s+([0-9]+(ns|us|µs|ms|s|m|h))+|([0-9*,/?-]+\s+){4}[0-9*,/?-]+)$`
	Schedules []string `json:"schedules"`
	// When is an expression that determines if a run should be scheduled.
	When string `json:"when,omitempty"`
}

// OrderDefaults is used to set defaults for all other artifacts of an Order.
type OrderDefaults struct {
	// SrcRef defines which Endpoint object is used as fallback source by all artifacts.
	// +optional
	SrcRef corev1.LocalObjectReference `json:"srcRef,omitempty"`
	// DstRef defines which Endpoint object is used as fallback destination by all artifacts.
	// +optional
	DstRef corev1.LocalObjectReference `json:"dstRef,omitempty"`
	// Cron specifies options which determine when the order should be scheduled.
	// +optional
	Cron OrderCron `json:"cron"`
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
	// Cron specifies options which determine when the order should be scheduled (falls back to OrderDefaults).
	// +optional
	Cron OrderCron `json:"cron"`
}

// OrderSpec defines the desired state of Order
type OrderSpec struct {
	// Defaults sets up defaults for all artifacts.
	// +optional
	Defaults OrderDefaults `json:"defaults,omitempty"`
	// Artifacts lists all artifacts, that will be processed by this Order.
	Artifacts []OrderArtifact `json:"artifacts,omitempty"`
	// +optional
	// TTLSecondsAfterCompletion specifies the time to live for the created ArtifactWorkflow(s) after completion.
	// After this time, the ArtifactWorkflow(s) are automatically deleted.
	// If unset, the ArtifactWorkflow(s) are automatically deleted immediately after completion.
	TTLSecondsAfterCompletion *int64 `json:"TTLSecondsAfterCompletion,omitempty"`
}

// OrderStatus defines the observed state of Order
type OrderStatus struct {
	// ArtifactWorkflows tracks the created workflows
	ArtifactWorkflows map[string]OrderArtifactWorkflowStatus `json:"artifactWorkflows,omitempty"`
	// A human readable message describing the current condition of the order.
	Message string `json:"message,omitempty"`
	// LastReconcileAt is the last time the Order was reconciled
	LastReconcileAt metav1.Time `json:"lastReconcileAt,omitempty"`
	// LastForceAt is the last time a force reconciliation was requested
	LastForceAt metav1.Time `json:"lastForceAt,omitempty"`
}

type OrderArtifactWorkflowStatus struct {
	// ArtifactIndex references back the index the corresponding artifact has in the .Spec
	ArtifactIndex int `json:"artifactIndex"`
	// Phase tracks which phase the corresponding Workflow is in
	Phase WorkflowPhase `json:"phase"`
	// A human readable message describing the current condition of the artifact workflow.
	Message string `json:"message,omitempty"`
	// CompletionTime is the time when the workflow finished
	CompletionTime metav1.Time `json:"completionTime,omitempty"`
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
