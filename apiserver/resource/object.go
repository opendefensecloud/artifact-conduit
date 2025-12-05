// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// Object is the core interface for all internal resource types stored in the API server.
// It provides metadata access, scoping, and factory methods for resource and list creation.
type Object interface {
	// All Objects must also be runtime.Object (for deep copy, type info, etc).
	runtime.Object

	// GetObjectMeta returns the object meta reference.
	GetObjectMeta() *metav1.ObjectMeta

	// Scoper defines whether the object is namespace-scoped or cluster-scoped.
	rest.Scoper

	// New returns a new instance of the resource -- e.g. &v1.Pod{}
	New() runtime.Object

	// NewList return a new list instance of the resource -- e.g. &v1.PodList{}
	NewList() runtime.Object

	// GetGroupResource returns the GroupResource for this object. The resource should
	// be the all lowercase and pluralized kind.
	GetGroupResource() schema.GroupResource
}

// ObjectWithDeepCopy is an optional extension for objects that support deep copying into another instance.
// E is the concrete type implementing Object.
type ObjectWithDeepCopy[E Object] interface {
	Object

	// DeepCopyInto copies the receiver's data into the provided obj.
	DeepCopyInto(obj E)
}

// ObjectWithStatusSubResource is implemented by resources that have a status subresource.
// It allows copying status fields between objects, useful for update strategies.
type ObjectWithStatusSubResource interface {
	Object

	// CopyStatusTo copies the status fields from the receiver to the target object.
	// Used to preserve status on updates where only spec changes are allowed.
	CopyStatusTo(runtime.Object)
}
