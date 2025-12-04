package rest

import (
	"fmt"

	"go.opendefense.cloud/arc/apiserver/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

type Storage = rest.Storage

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not a resource.Object
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	provider, ok := obj.(resource.Object)
	if !ok {
		return nil, nil, fmt.Errorf("given object of type %T does not have metadata", obj)
	}
	om := provider.GetObjectMeta()
	return om.GetLabels(), SelectableFields(om), nil
}

// SelectableFields returns a field set that represents the object.
func SelectableFields(obj *metav1.ObjectMeta) fields.Set {
	return generic.ObjectMetaFieldsSet(obj, true)
}

func NewStore(
	scheme *runtime.Scheme,
	single, list func() runtime.Object,
	gr schema.GroupResource,
	strategy Strategy, optsGetter generic.RESTOptionsGetter) (*genericregistry.Store, error) {
	store := &genericregistry.Store{
		NewFunc:                   single,
		NewListFunc:               list,
		PredicateFunc:             strategy.Match,
		DefaultQualifiedResource:  gr,
		SingularQualifiedResource: gr,
		TableConvertor:            strategy,
		CreateStrategy:            strategy,
		UpdateStrategy:            strategy,
		DeleteStrategy:            strategy,
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return store, nil
}
