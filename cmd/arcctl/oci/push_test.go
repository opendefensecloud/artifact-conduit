// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !release

package oci

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var _ = Describe("Push Command", func() {
	var (
		cmd *cobra.Command
	)
	const (
		arcctlTempDir = "/tmp/arcctl-tests"
	)

	BeforeEach(func() {
		cmd = NewPushCommand()
		cmd.SetContext(GinkgoT().Context())
	})

	AfterEach(func() {
		viper.Reset()
		Expect(os.RemoveAll(arcctlTempDir)).ToNot(HaveOccurred())
	})

	Context("when required configuration is missing", func() {
		It("should return an error if tmp-dir is missing", func() {
			err := runPush(cmd, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tmp-dir is not set"))
		})
	})

})
