// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/test/oci"
)

var (
	registry oci.Registry
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCI Suite")
	registry = *oci.NewRegistry()
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
})
