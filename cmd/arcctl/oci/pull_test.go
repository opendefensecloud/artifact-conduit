// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var _ = Describe("Pull Command", func() {
	var (
		cmd *cobra.Command
	)
	const (
		arcctlTempDir = "/tmp/arcctl-tests"
	)

	BeforeEach(func() {
		cmd = NewPullCommand()
		cmd.SetContext(GinkgoT().Context())
	})

	AfterEach(func() {
		viper.Reset()
		Expect(os.RemoveAll(arcctlTempDir)).ToNot(HaveOccurred())
	})

	Context("when required configuration is missing", func() {
		It("should return an error if source.reference is missing", func() {
			err := runPull(cmd, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source.reference is not set"))
		})
	})

	Context("when configuration is valid", func() {
		BeforeEach(func() {
			viper.Set("source.reference", "registry-1.docker.io/library/busybox:latest")
			viper.Set("tmp-dir", arcctlTempDir)
		})

		It("should pull the OCI artifact successfully", func() {
			err := runPull(cmd, []string{})
			Expect(err).ToNot(HaveOccurred())
			// Verify oci layout on disk exists
			_, err = os.Stat(arcctlTempDir + "/index.json")
			Expect(err).ToNot(HaveOccurred())
		})

	})
})
