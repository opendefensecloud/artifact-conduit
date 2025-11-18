// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !release

package oci

import (
	"bytes"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opendefense.cloud/arc/pkg/workflow/config"
	ocitest "go.opendefense.cloud/arc/test/oci"
)

var _ = Describe("Pull Command", func() {
	const (
		arcctlTempDir = "/tmp/arcctl-tests"
	)
	var (
		cmd *cobra.Command
	)

	BeforeEach(func() {
		cmd = NewPullCommand()
		cmd.SetContext(GinkgoT().Context())
		viper.SetConfigType("json")
		viper.Set("tmp-dir", arcctlTempDir)
		viper.Set("plain-http", true)
	})

	AfterEach(func() {
		viper.Reset()
		Expect(os.RemoveAll(arcctlTempDir)).ToNot(HaveOccurred())
	})

	Context("when configuration is invalid", func() {
		It("should fail to pull the OCI artifact", func() {
			// setup config
			conf := &config.WorkflowConfig{}

			confJson, err := conf.ToJson()
			Expect(err).ToNot(HaveOccurred())
			Expect(viper.ReadConfig(bytes.NewReader(confJson))).To(Succeed())

			// actually pull
			err = runPull(cmd, []string{})
			Expect(err).To(HaveOccurred())
		})
	})
	Context("when auth is necessary", func() {
		BeforeEach(func() {
			testEnv.Setup(cmd.Context())
			testEnv.EnableAuth()
		})
		AfterEach(func() {
			testEnv.Shutdown(cmd.Context())
		})

		It("should pull the OCI artifact successfully with auth set", func() {
			// setup config
			conf := &config.WorkflowConfig{}
			conf.Type = config.AT_OCI
			conf.Src.Type = config.AT_OCI
			conf.Src.RemoteURL = testEnv.MockRegistry.Listener.Addr().String()
			conf.Src.Auth = config.OCIAuth{
				Username: ocitest.AuthUser,
				Password: ocitest.AuthPass,
			}
			conf.Spec = config.OCISpec{
				Image: testEnv.MockReference,
			}

			confJson, err := conf.ToJson()
			Expect(err).ToNot(HaveOccurred())
			Expect(viper.ReadConfig(bytes.NewReader(confJson))).To(Succeed())

			// actually pull
			Expect(runPull(cmd, []string{})).To(Succeed())

			// verify oci layout on disk exists
			_, err = os.Stat(arcctlTempDir + "/index.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail to pull the OCI artifact successfully with auth missing", func() {
			// setup config
			conf := &config.WorkflowConfig{}
			conf.Type = config.AT_OCI
			conf.Src.Type = config.AT_OCI
			conf.Src.RemoteURL = testEnv.MockRegistry.Listener.Addr().String()
			conf.Spec = config.OCISpec{
				Image: testEnv.MockReference,
			}

			confJson, err := conf.ToJson()
			Expect(err).ToNot(HaveOccurred())
			Expect(viper.ReadConfig(bytes.NewReader(confJson))).To(Succeed())

			// actually pull
			err = runPull(cmd, []string{})
			Expect(err).To(HaveOccurred())
		})
	})
})
