// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/pkg/registry"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
)

func TestRegistry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Suite")
}

var _ = Describe("RESTInPeace", func() {
	It("should return storage when error is nil", func() {
		store := &registry.REST{Store: &genericregistry.Store{}}
		result := registry.RESTInPeace(store, nil)
		Expect(result).To(Equal(store))
	})

	It("should panic when error is not nil", func() {
		Expect(func() {
			registry.RESTInPeace(nil, errors.New("test error"))
		}).To(Panic())
	})

	It("should panic with wrapped error message", func() {
		defer func() {
			r := recover()
			Expect(r).NotTo(BeNil())
			err, ok := r.(error)
			Expect(ok).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("unable to create REST storage"))
			Expect(err.Error()).To(ContainSubstring("test error"))
		}()
		registry.RESTInPeace(nil, errors.New("test error"))
	})
})
