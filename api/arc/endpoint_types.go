// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc

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

// EndpointSpec defines the desired state of Endpoint.
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

// Endpoint condition types reported in EndpointStatus.Conditions.
const (
	// EndpointConditionValidated reports whether the Endpoint's configuration
	// resolves: its Secret exists and its type is claimed by an ArtifactType.
	EndpointConditionValidated = "Validated"
	// EndpointConditionReachable reports whether the target answered at all.
	// Any HTTP response counts, including 401.
	EndpointConditionReachable = "Reachable"
	// EndpointConditionAuthenticated reports whether the Endpoint's credentials
	// were accepted. Unknown means ARC cannot verify credentials of this shape,
	// which is not a failure.
	EndpointConditionAuthenticated = "Authenticated"
	// EndpointConditionReady summarises the others.
	EndpointConditionReady = "Ready"
)

// EndpointStatus defines the observed state of Endpoint
type EndpointStatus struct {
	// Conditions represent the latest available observations of the Endpoint's state.
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// ObservedGeneration is the .metadata.generation the conditions were computed from.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// LastProbeTime is when the connection to spec.remoteURL was last attempted.
	// +optional
	LastProbeTime *metav1.Time `json:"lastProbeTime,omitempty"`
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
