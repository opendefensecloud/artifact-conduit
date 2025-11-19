// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EndpointUsage is the usage of the endpoint.
// +enum
type EndpointUsage string

const (
	// EndpointUsagePullOnly means the endpoint can only be used to pull data.
	EndpointUsagePullOnly EndpointUsage = "PullOnly"
	// EndpointUsagePushOnly means the endpoint can only be used to push data.
	EndpointUsagePushOnly EndpointUsage = "PushOnly"
	// EndpointUsageAll means the endpoint can be used with all kinds of usage patterns.
	EndpointUsageAll EndpointUsage = "All"
)

// EndpointSpec specifies a single artifact which is translated into a corresponding Workflow based on its type.
type EndpointSpec struct {
	// Type specifies which ArtifactType is used to process this artifact.
	Type string `json:"type"`
	// RemoteURL defines the URL which is used to interact with the endpoint.
	RemoteURL string `json:"remoteURL"`
	// SecretRef specifies the secret containing the relevant credentials for the endpoint.
	// +optional
	SecretRef corev1.LocalObjectReference `json:"secretRef"`
	// Usage defines how the endpoint is allowed to be used.
	Usage EndpointUsage `json:"usage"`
}

// EndpointStatus defines the observed state of Endpoint
type EndpointStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Endpoint is the Schema for the endpoints API
type Endpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   EndpointSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status EndpointStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EndpointList is a list of Endpoint objects.
type EndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Endpoint `json:"items" protobuf:"bytes,2,rep,name=items"`
}
