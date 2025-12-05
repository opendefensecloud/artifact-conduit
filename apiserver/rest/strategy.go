// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

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

// Strategy defines the set of hooks and behaviors used by the API server for resource storage operations.
// It combines create, update, delete, and table conversion strategies, plus a predicate matcher for filtering.
type Strategy interface {
	// Match returns a predicate for filtering resources by label and field selectors.
	Match(label labels.Selector, field fields.Selector) storage.SelectionPredicate
	rest.RESTUpdateStrategy
	rest.RESTCreateStrategy
	rest.RESTDeleteStrategy
	rest.TableConvertor
}

var _ Strategy = DefaultStrategy{}

// DefaultStrategy is a generic implementation of Strategy.
// It delegates most behaviors to interfaces implemented by the underlying Object, if present.
// If the Object does not implement an override interface, DefaultStrategy provides a fallback.
type DefaultStrategy struct {
	// Object is the resource instance whose interfaces may override default behaviors.
	Object runtime.Object
	// ObjectTyper provides type information for the resource.
	runtime.ObjectTyper
	// TableConvertor is used for table output if the object does not implement TableConverter.
	TableConvertor rest.TableConvertor
}

// NewDefaultStrategy constructs a DefaultStrategy for a given resource type.
// obj: a sample instance of the resource
// objTyper: type information provider
// gr: group/resource descriptor for table conversion
func NewDefaultStrategy(obj runtime.Object, objTyper runtime.ObjectTyper, gr schema.GroupResource) *DefaultStrategy {
	return &DefaultStrategy{
		Object:         obj,
		ObjectTyper:    objTyper,
		TableConvertor: rest.NewDefaultTableConvertor(gr),
	}
}

// GenerateName returns a generated name for a resource, using the object's NameGenerator if present.
func (d DefaultStrategy) GenerateName(base string) string {
	if d.Object == nil {
		return names.SimpleNameGenerator.GenerateName(base)
	}
	if n, ok := d.Object.(NameGenerator); ok {
		return n.GenerateName(base)
	}
	return names.SimpleNameGenerator.GenerateName(base)
}

// NamespaceScoped returns true if the resource is namespaced, using the object's Scoper if present.
func (d DefaultStrategy) NamespaceScoped() bool {
	if d.Object == nil {
		return true
	}
	if n, ok := d.Object.(Scoper); ok {
		return n.NamespaceScoped()
	}
	return true
}

// PrepareForCreate normalizes the object before creation, delegating to PrepareForCreater if implemented.
func (DefaultStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	if v, ok := obj.(PrepareForCreater); ok {
		v.PrepareForCreate(ctx)
	}
}

// PrepareForUpdate normalizes the object before update.
// If the object has a status subresource, status is copied from old to new.
// If PrepareForUpdater is implemented, it is called to further normalize.
func (DefaultStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	if v, ok := obj.(resource.ObjectWithStatusSubResource); ok {
		// Copy status from old to new to avoid spec-only updates modifying status.
		old.(resource.ObjectWithStatusSubResource).CopyStatusTo(v)
	}
	if v, ok := obj.(PrepareForUpdater); ok {
		v.PrepareForUpdate(ctx, old)
	}
}

// Validate delegates to the object's Validater interface if present, otherwise returns no errors.
func (DefaultStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	if v, ok := obj.(Validater); ok {
		return v.Validate(ctx)
	}
	return field.ErrorList{}
}

// AllowCreateOnUpdate returns true if the object allows creation via update (PUT), using AllowCreateOnUpdater if present.
func (d DefaultStrategy) AllowCreateOnUpdate() bool {
	if d.Object == nil {
		return false
	}
	if n, ok := d.Object.(AllowCreateOnUpdater); ok {
		return n.AllowCreateOnUpdate()
	}
	return false
}

// AllowUnconditionalUpdate returns true if the object allows unconditional updates, using AllowUnconditionalUpdater if present.
func (d DefaultStrategy) AllowUnconditionalUpdate() bool {
	if d.Object == nil {
		return false
	}
	if n, ok := d.Object.(AllowUnconditionalUpdater); ok {
		return n.AllowUnconditionalUpdate()
	}
	return false
}

// Canonicalize mutates the object into a canonical form if Canonicalizer is implemented.
func (DefaultStrategy) Canonicalize(obj runtime.Object) {
	if c, ok := obj.(Canonicalizer); ok {
		c.Canonicalize()
	}
}

// ValidateUpdate delegates to the object's ValidateUpdater interface if present, otherwise returns no errors.
func (DefaultStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	if v, ok := obj.(ValidateUpdater); ok {
		return v.ValidateUpdate(ctx, old)
	}
	return field.ErrorList{}
}

// Match returns a SelectionPredicate for filtering resources by label and field selectors.
func (DefaultStrategy) Match(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// ConvertToTable returns a Table representation of the object, using TableConverter if implemented.
func (d DefaultStrategy) ConvertToTable(
	ctx context.Context, obj runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	if c, ok := obj.(TableConverter); ok {
		return c.ConvertToTable(ctx, tableOptions)
	}
	return d.TableConvertor.ConvertToTable(ctx, obj, tableOptions)
}

// WarningsOnCreate returns any warnings for create operations (default: none).
func (d DefaultStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// WarningsOnUpdate returns any warnings for update operations (default: none).
func (d DefaultStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// PrepareForUpdaterStrategy is a wrapper for RESTUpdateStrategy that allows custom update normalization via OverrideFn.
type PrepareForUpdaterStrategy struct {
	rest.RESTUpdateStrategy
	// OverrideFn is called to perform custom normalization during update.
	OverrideFn func(ctx context.Context, obj, old runtime.Object)
}

// PrepareForUpdate calls the custom OverrideFn if set, otherwise does nothing.
func (s *PrepareForUpdaterStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	if s.OverrideFn != nil {
		s.OverrideFn(ctx, obj, old)
	}
}
