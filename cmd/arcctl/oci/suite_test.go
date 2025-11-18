// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !release

package oci

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/test/oci"
)

var (
	testEnv *oci.TestEnv
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	testEnv = oci.NewTestEnv()
	RunSpecs(t, "OCI Suite")
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
})
