package rest

import (
	"context"

	"go.opendefense.cloud/arc/apiserver/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
)

// Strategy defines functions that are invoked prior to storing a Kubernetes resource.
type Strategy interface {
	Match(label labels.Selector, field fields.Selector) storage.SelectionPredicate
	rest.RESTUpdateStrategy
	rest.RESTCreateStrategy
	rest.RESTDeleteStrategy
	rest.TableConvertor
}

var _ Strategy = DefaultStrategy{}

// DefaultStrategy implements Strategy. DefaultStrategy will delegate to functions
// specified on the underlying Object.
// DefaultStrategy provides the default used by all Objects as a fallback.
type DefaultStrategy struct {
	Object runtime.Object
	runtime.ObjectTyper
	TableConvertor rest.TableConvertor
}

func NewDefaultStrategy(obj runtime.Object, objTyper runtime.ObjectTyper, gr schema.GroupResource) *DefaultStrategy {
	return &DefaultStrategy{
		Object:         obj,
		ObjectTyper:    objTyper,
		TableConvertor: rest.NewDefaultTableConvertor(gr),
	}
}

// GenerateName generates a new name for a resource without one.
func (d DefaultStrategy) GenerateName(base string) string {
	if d.Object == nil {
		return names.SimpleNameGenerator.GenerateName(base)
	}
	if n, ok := d.Object.(NameGenerator); ok {
		return n.GenerateName(base)
	}
	return names.SimpleNameGenerator.GenerateName(base)
}

// NamespaceScoped is used to register the resource as namespaced or non-namespaced.
func (d DefaultStrategy) NamespaceScoped() bool {
	if d.Object == nil {
		return true
	}
	if n, ok := d.Object.(Scoper); ok {
		return n.NamespaceScoped()
	}
	return true
}

// PrepareForCreate calls the PrepareForCreate function on obj if supported, otherwise does nothing.
func (DefaultStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	if v, ok := obj.(PrepareForCreater); ok {
		v.PrepareForCreate(ctx)
	}
}

// PrepareForUpdate calls the PrepareForUpdate function on obj if supported, otherwise does nothing.
func (DefaultStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	if v, ok := obj.(resource.ObjectWithStatusSubResource); ok {
		// don't modify the status
		old.(resource.ObjectWithStatusSubResource).CopyStatusTo(v)
	}
	if v, ok := obj.(PrepareForUpdater); ok {
		v.PrepareForUpdate(ctx, old)
	}
}

func (DefaultStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	if v, ok := obj.(Validater); ok {
		return v.Validate(ctx)
	}
	return field.ErrorList{}
}

func (d DefaultStrategy) AllowCreateOnUpdate() bool {
	if d.Object == nil {
		return false
	}
	if n, ok := d.Object.(AllowCreateOnUpdater); ok {
		return n.AllowCreateOnUpdate()
	}
	return false
}

func (d DefaultStrategy) AllowUnconditionalUpdate() bool {
	if d.Object == nil {
		return false
	}
	if n, ok := d.Object.(AllowUnconditionalUpdater); ok {
		return n.AllowUnconditionalUpdate()
	}
	return false
}

func (DefaultStrategy) Canonicalize(obj runtime.Object) {
	if c, ok := obj.(Canonicalizer); ok {
		c.Canonicalize()
	}
}

func (DefaultStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	if v, ok := obj.(ValidateUpdater); ok {
		return v.ValidateUpdate(ctx, old)
	}
	return field.ErrorList{}
}

func (DefaultStrategy) Match(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func (d DefaultStrategy) ConvertToTable(
	ctx context.Context, obj runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	if c, ok := obj.(TableConverter); ok {
		return c.ConvertToTable(ctx, tableOptions)
	}
	return d.TableConvertor.ConvertToTable(ctx, obj, tableOptions)
}

func (d DefaultStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (d DefaultStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type PrepareForUpdaterStrategy struct {
	rest.RESTUpdateStrategy
	OverrideFn func(ctx context.Context, obj, old runtime.Object)
}

func (s *PrepareForUpdaterStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	s.OverrideFn(ctx, obj, old)
}
