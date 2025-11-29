// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package apiserver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/pkg/apiserver"
	"k8s.io/apimachinery/pkg/util/version"
)

var _ = Describe("ARCVersionToKubeVersion", func() {
	It("should return nil for major version not equal to 1", func() {
		ver := version.MustParse("2.0")
		result := apiserver.ARCVersionToKubeVersion(ver)
		Expect(result).To(BeNil())
	})

	It("should return nil for major version 0", func() {
		ver := version.MustParse("0.1")
		result := apiserver.ARCVersionToKubeVersion(ver)
		Expect(result).To(BeNil())
	})

	It("should map version 1.2 correctly", func() {
		ver := version.MustParse("1.2")
		result := apiserver.ARCVersionToKubeVersion(ver)
		Expect(result).NotTo(BeNil())
		// 1.2 should map to the default kube version (offset 0)
	})

	It("should map version 1.1 to kube version with offset -1", func() {
		ver := version.MustParse("1.1")
		result := apiserver.ARCVersionToKubeVersion(ver)
		Expect(result).NotTo(BeNil())
		// 1.1 should be one minor version below the default kube version
	})

	It("should map version 1.0 to kube version with offset -2", func() {
		ver := version.MustParse("1.0")
		result := apiserver.ARCVersionToKubeVersion(ver)
		Expect(result).NotTo(BeNil())
		// 1.0 should be two minor versions below the default kube version
	})

	It("should cap at current kube version for very high minor version", func() {
		ver := version.MustParse("1.100")
		result := apiserver.ARCVersionToKubeVersion(ver)
		Expect(result).NotTo(BeNil())
		// Should not exceed the current kube version
	})
})
