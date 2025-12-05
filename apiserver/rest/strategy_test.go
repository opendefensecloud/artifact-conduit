// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// testObj is a small helper type used to implement several of the
// optional interfaces that DefaultStrategy looks for.
type testObj struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Status string
	Flag   bool
}

func (t *testObj) DeepCopyObject() runtime.Object {
	if t == nil {
		return nil
	}
	copy := *t
	return &copy
}

func (t *testObj) GetObjectMeta() *metav1.ObjectMeta { return &t.ObjectMeta }
func (t *testObj) NamespaceScoped() bool             { return true }
func (t *testObj) New() runtime.Object               { return &testObj{} }
func (t *testObj) NewList() runtime.Object           { return &testObjList{} }

func (t *testObj) GetGroupResource() schema.GroupResource {
	return schema.GroupResource{Group: "arc", Resource: "testobjs"}
}

// CopyStatusTo implements resource.ObjectWithStatusSubResource
func (t *testObj) CopyStatusTo(obj runtime.Object) {
	if o, ok := obj.(*testObj); ok {
		o.Status = t.Status
	}
}

// PrepareForCreate implements PrepareForCreater
func (t *testObj) PrepareForCreate(ctx context.Context) { t.Flag = true }

// PrepareForUpdate implements PrepareForUpdater
func (t *testObj) PrepareForUpdate(ctx context.Context, old runtime.Object) { t.Flag = true }

// Validate implements Validater
func (t *testObj) Validate(ctx context.Context) field.ErrorList {
	return field.ErrorList{field.Invalid(field.NewPath("spec"), "bad", "invalid")}
}

// ValidateUpdate implements ValidateUpdater
func (t *testObj) ValidateUpdate(ctx context.Context, old runtime.Object) field.ErrorList {
	return field.ErrorList{field.Invalid(field.NewPath("spec"), "bad", "invalid")}
}

// Canonicalize implements Canonicalizer
func (t *testObj) Canonicalize() { t.Flag = true }

// ConvertToTable implements TableConverter
func (t *testObj) ConvertToTable(ctx context.Context, _ runtime.Object) (*metav1.Table, error) {
	return &metav1.Table{Rows: []metav1.TableRow{{}}}, nil
}

// testObjList is a minimal list type returned by NewList.
type testObjList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []testObj
}

func (t *testObjList) DeepCopyObject() runtime.Object {
	if t == nil {
		return nil
	}
	copy := *t
	return &copy
}

// nameGen implements NameGenerator
type nameGen struct {
	testObj
}

func (n *nameGen) GenerateName(base string) string { return base + "-GEN" }

// scoper implements Scoper
type scoper struct {
	testObj
}

func (s *scoper) NamespaceScoped() bool { return false }

// allowCreate implements AllowCreateOnUpdater
type allowCreate struct {
	testObj
}

func (a *allowCreate) AllowCreateOnUpdate() bool { return true }

// allowUnconditional implements AllowUnconditionalUpdater
type allowUnconditional struct {
	testObj
}

func (a *allowUnconditional) AllowUnconditionalUpdate() bool { return true }

var _ = Describe("DefaultStrategy", func() {
	It("should use NameGenerator for GenerateName", func() {
		ds := DefaultStrategy{Object: &nameGen{}}
		Expect(ds.GenerateName("base")).To(Equal("base-GEN"))
	})

	It("should use Scoper for NamespaceScoped", func() {
		ds := DefaultStrategy{Object: &scoper{}}
		Expect(ds.NamespaceScoped()).To(BeFalse())
	})

	It("should call PrepareForCreater on PrepareForCreate", func() {
		obj := &testObj{}
		ds := DefaultStrategy{}
		ds.PrepareForCreate(context.Background(), obj)
		Expect(obj.Flag).To(BeTrue())
	})

	It("should copy status and call PrepareForUpdater on PrepareForUpdate", func() {
		old := &testObj{Status: "old-status"}
		obj := &testObj{Status: "new-status"}
		ds := DefaultStrategy{}
		ds.PrepareForUpdate(context.Background(), obj, old)
		Expect(obj.Status).To(Equal("old-status"))
		Expect(obj.Flag).To(BeTrue())
	})

	It("should delegate Validate and ValidateUpdate to object", func() {
		obj := &testObj{}
		ds := DefaultStrategy{}
		Expect(ds.Validate(context.Background(), obj)).ToNot(BeEmpty())
		Expect(ds.ValidateUpdate(context.Background(), obj, &testObj{})).ToNot(BeEmpty())
	})

	It("should delegate AllowCreateOnUpdate and AllowUnconditionalUpdate", func() {
		ds1 := DefaultStrategy{Object: &allowCreate{}}
		Expect(ds1.AllowCreateOnUpdate()).To(BeTrue())
		ds2 := DefaultStrategy{Object: &allowUnconditional{}}
		Expect(ds2.AllowUnconditionalUpdate()).To(BeTrue())
	})

	It("should delegate Canonicalize and ConvertToTable to object", func() {
		obj := &testObj{}
		ds := DefaultStrategy{Object: obj}
		ds.Canonicalize(obj)
		Expect(obj.Flag).To(BeTrue())
		tbl, err := ds.ConvertToTable(context.Background(), obj, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(tbl).ToNot(BeNil())
		Expect(tbl.Rows).To(HaveLen(1))
	})
})

var _ = Describe("PrepareForUpdaterStrategy", func() {
	It("should call OverrideFn on PrepareForUpdate", func() {
		called := false
		var gotCtx context.Context
		var gotObj, gotOld runtime.Object
		s := &PrepareForUpdaterStrategy{
			RESTUpdateStrategy: &DefaultStrategy{Object: &testObj{}},
			OverrideFn: func(ctx context.Context, obj, old runtime.Object) {
				called = true
				gotCtx = ctx
				gotObj = obj
				gotOld = old
			},
		}
		obj := &testObj{Status: "new"}
		old := &testObj{Status: "old"}
		//nolint:staticcheck
		ctx := context.WithValue(context.Background(), "key", "val")
		s.PrepareForUpdate(ctx, obj, old)
		Expect(called).To(BeTrue())
		Expect(gotCtx).To(Equal(ctx))
		Expect(gotObj).To(Equal(obj))
		Expect(gotOld).To(Equal(old))
	})

	It("should not panic if OverrideFn is nil", func() {
		s := &PrepareForUpdaterStrategy{}
		obj := &testObj{}
		old := &testObj{}
		// Should not panic, but does nothing
		Expect(func() { s.PrepareForUpdate(context.Background(), obj, old) }).ToNot(Panic())
	})
})
