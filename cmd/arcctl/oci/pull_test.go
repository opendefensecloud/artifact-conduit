// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"bytes"
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opendefense.cloud/arc/pkg/workflow/config"
	"go.opendefense.cloud/arc/test/oci"
)

var _ = Describe("Pull Command", func() {
	const (
		arcctlTempDir       = "/tmp/arcctl-tests"
		mockImageRepository = "arc"
		mockImageName       = "test"
		mockImageVersion    = "v0.0.0"
	)
	var (
		ctx           context.Context
		cmd           *cobra.Command
		mockRegistry  *oci.Registry
		mockReference string
	)

	BeforeEach(func() {
		ctx = context.Background()
		cmd = NewPullCommand()
		cmd.SetContext(GinkgoT().Context())
		viper.SetConfigType("json")
		viper.Set("tmp-dir", arcctlTempDir)
		viper.Set("plain-http", true)

		mockRegistry = oci.NewRegistry()
		mockRepository := fmt.Sprintf("%s/%s/%s", mockRegistry.Listener.Addr().String(), mockImageRepository, mockImageName)

		mockReference = fmt.Sprintf("%s/%s:%s", mockImageRepository, mockImageName, mockImageVersion)
		_, err := oci.PushTestManifest(ctx, mockRepository, mockImageVersion)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		viper.Reset()
		Expect(os.RemoveAll(arcctlTempDir)).ToNot(HaveOccurred())
		mockRegistry.Close()

	})

	Context("when configuration is valid", func() {
		It("should pull the OCI artifact successfully", func() {
			conf := &config.ArcctlConfig{}
			conf.Type = config.AT_OCI
			conf.Src.Type = config.AT_OCI
			conf.Src.RemoteURL = mockRegistry.Listener.Addr().String()
			conf.Spec = config.OCISpec{
				Image: mockReference,
			}

			confJson, err := conf.ToJson()
			Expect(err).ToNot(HaveOccurred())
			Expect(viper.ReadConfig(bytes.NewReader(confJson))).To(Succeed())

			Expect(runPull(cmd, []string{})).To(Succeed())
			// Verify oci layout on disk exists
			_, err = os.Stat(arcctlTempDir + "/index.json")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
