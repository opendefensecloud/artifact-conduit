package resource

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// Object must be implemented by all internal versions of resources that are stored.
type Object interface {
	// All Objects must also be runtime.Object.
	runtime.Object

	// GetObjectMeta returns the object meta reference.
	GetObjectMeta() *metav1.ObjectMeta

	// Scoper is used to define the objects scope, either namespace scoped or non-namespace scoped.
	rest.Scoper

	// New returns a new instance of the resource -- e.g. &v1.Pod{}
	New() runtime.Object

	// NewList return a new list instance of the resource -- e.g. &v1.PodList{}
	NewList() runtime.Object

	// GetGroupResource returns the GroupResource for this object. The resource should
	// be the all lowercase and pluralized kind.
	GetGroupResource() schema.GroupResource
}

type ObjectWithDeepCopy[E Object] interface {
	Object

	DeepCopyInto(obj E)
}

type ObjectWithStatusSubResource interface {
	Object

	CopyStatusTo(runtime.Object)
}
