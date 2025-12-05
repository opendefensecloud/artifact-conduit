// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package apiserver

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.opendefense.cloud/arc/apiserver/rest"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("mergeVersionedResourcesStorageMap", func() {
	It("should merge two empty maps", func() {
		a := map[string]map[string]rest.Storage{}
		b := map[string]map[string]rest.Storage{}
		result := mergeVersionedResourcesStorageMap(a, b)
		Expect(result).To(BeEmpty())
	})

	It("should return copy of a when b is empty", func() {
		a := map[string]map[string]rest.Storage{
			"v1": {
				"pods":     nil,
				"services": nil,
			},
		}
		b := map[string]map[string]rest.Storage{}
		result := mergeVersionedResourcesStorageMap(a, b)
		Expect(result).To(HaveKey("v1"))
		Expect(result["v1"]).To(HaveLen(2))
		Expect(result["v1"]).To(HaveKey("pods"))
		Expect(result["v1"]).To(HaveKey("services"))
	})

	It("should return copy of b when a is empty", func() {
		a := map[string]map[string]rest.Storage{}
		b := map[string]map[string]rest.Storage{
			"v1beta1": {
				"orders": nil,
			},
		}
		result := mergeVersionedResourcesStorageMap(a, b)
		Expect(result).To(HaveKey("v1beta1"))
		Expect(result["v1beta1"]).To(HaveLen(1))
		Expect(result["v1beta1"]).To(HaveKey("orders"))
	})

	It("should merge resources from different versions", func() {
		a := map[string]map[string]rest.Storage{
			"v1": {
				"pods": nil,
			},
		}
		b := map[string]map[string]rest.Storage{
			"v1beta1": {
				"orders": nil,
			},
		}
		result := mergeVersionedResourcesStorageMap(a, b)
		Expect(result).To(HaveLen(2))
		Expect(result).To(HaveKey("v1"))
		Expect(result).To(HaveKey("v1beta1"))
		Expect(result["v1"]).To(HaveKey("pods"))
		Expect(result["v1beta1"]).To(HaveKey("orders"))
	})

	It("should merge resources from the same version", func() {
		a := map[string]map[string]rest.Storage{
			"v1": {
				"pods": nil,
			},
		}
		b := map[string]map[string]rest.Storage{
			"v1": {
				"services": nil,
			},
		}
		result := mergeVersionedResourcesStorageMap(a, b)
		Expect(result).To(HaveKey("v1"))
		Expect(result["v1"]).To(HaveLen(2))
		Expect(result["v1"]).To(HaveKey("pods"))
		Expect(result["v1"]).To(HaveKey("services"))
	})

	It("should preserve storage references from both maps", func() {
		storageA := &mockStorage{name: "storageA"}
		storageB := &mockStorage{name: "storageB"}
		a := map[string]map[string]rest.Storage{
			"v1": {
				"pods": storageA,
			},
		}
		b := map[string]map[string]rest.Storage{
			"v1": {
				"services": storageB,
			},
		}
		result := mergeVersionedResourcesStorageMap(a, b)
		Expect(result["v1"]["pods"]).To(Equal(storageA))
		Expect(result["v1"]["services"]).To(Equal(storageB))
	})

	It("should handle multiple versions and resources", func() {
		a := map[string]map[string]rest.Storage{
			"v1": {
				"pods":     nil,
				"services": nil,
			},
			"v1beta1": {
				"orders": nil,
			},
		}
		b := map[string]map[string]rest.Storage{
			"v1": {
				"nodes": nil,
			},
			"v1alpha1": {
				"fragments": nil,
			},
		}
		result := mergeVersionedResourcesStorageMap(a, b)
		Expect(result).To(HaveLen(3))
		Expect(result["v1"]).To(HaveLen(3))
		Expect(result["v1beta1"]).To(HaveLen(1))
		Expect(result["v1alpha1"]).To(HaveLen(1))
		Expect(result["v1"]).To(HaveKey("pods"))
		Expect(result["v1"]).To(HaveKey("services"))
		Expect(result["v1"]).To(HaveKey("nodes"))
		Expect(result["v1beta1"]).To(HaveKey("orders"))
		Expect(result["v1alpha1"]).To(HaveKey("fragments"))
	})

	It("should not modify input maps", func() {
		a := map[string]map[string]rest.Storage{
			"v1": {
				"pods": nil,
			},
		}
		b := map[string]map[string]rest.Storage{
			"v1": {
				"services": nil,
			},
		}
		// Keep copies to compare later
		aOriginal := map[string]map[string]rest.Storage{
			"v1": {
				"pods": nil,
			},
		}
		bOriginal := map[string]map[string]rest.Storage{
			"v1": {
				"services": nil,
			},
		}
		_ = mergeVersionedResourcesStorageMap(a, b)
		// Verify inputs weren't modified
		Expect(a).To(Equal(aOriginal))
		Expect(b).To(Equal(bOriginal))
	})

	It("should handle deeply nested structures", func() {
		a := map[string]map[string]rest.Storage{
			"v1": {
				"resource1": nil,
				"resource2": nil,
			},
			"v1beta1": {
				"resource3": nil,
			},
		}
		b := map[string]map[string]rest.Storage{
			"v1": {
				"resource4": nil,
			},
			"v1beta1": {
				"resource5": nil,
				"resource6": nil,
			},
			"v2": {
				"resource7": nil,
			},
		}
		result := mergeVersionedResourcesStorageMap(a, b)
		Expect(result).To(HaveLen(3))
		Expect(result["v1"]).To(HaveLen(3))
		Expect(result["v1beta1"]).To(HaveLen(3))
		Expect(result["v2"]).To(HaveLen(1))
	})
})

// mockStorage is a minimal implementation of rest.Storage for testing.
type mockStorage struct {
	name string
}

func (m *mockStorage) New() runtime.Object {
	return &mockStorage{name: m.name}
}

func (m *mockStorage) Destroy() {}

// DeepCopyObject implements runtime.Object interface.
func (m *mockStorage) DeepCopyObject() runtime.Object {
	if m == nil {
		return nil
	}
	copy := *m
	return &copy
}

// GetObjectKind implements runtime.Object interface.
func (m *mockStorage) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}
