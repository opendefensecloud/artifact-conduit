// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// FragmentSpec specifies a single artifact which is translated into a corresponding Workflow based on its type.
type FragmentSpec struct {
	// Type specifies which ArtifactTypeDefinition is used to process this artifact.
	Type string `json:"type"`
	// SrcRef defines which Endpoint object is used as source.
	SrcRef corev1.LocalObjectReference `json:"srcRef"`
	// SrcRef defines which Endpoint object is used as destination.
	DstRef corev1.LocalObjectReference `json:"dstRef"`
	// Spec specifies parameters used by the underlying Workflow.
	Spec runtime.RawExtension `json:"spec,omitempty"`
}

// FragmentStatus defines the observed state of Fragment
type FragmentStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Fragment is the Schema for the fragments API
type Fragment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   FragmentSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status FragmentStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FragmentList is a list of Fragment objects.
type FragmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Fragment `json:"items" protobuf:"bytes,2,rep,name=items"`
}
