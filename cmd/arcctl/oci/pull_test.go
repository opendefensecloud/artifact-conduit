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
	ocitest "go.opendefense.cloud/arc/test/oci"
)

var _ = Describe("Pull Command", func() {
	const (
		arcctlTempDir       = "/tmp/arcctl-tests"
		mockImageRepository = "arc"
		mockImageName       = "test"
		mockImageVersion    = "v0.0.0"
		authuser            = "user"
		authpass            = "pass"
	)
	var (
		ctx           context.Context
		cmd           *cobra.Command
		mockRegistry  *ocitest.Registry
		mockReference string
	)

	setupRegistry := func(needsAuth bool) {
		if needsAuth {
			mockRegistry = ocitest.NewRegistry().WithAuth(authuser, authpass)
		} else {
			mockRegistry = ocitest.NewRegistry()
		}

		repo := fmt.Sprintf("%s/%s/%s", mockRegistry.Listener.Addr().String(), mockImageRepository, mockImageName)
		mockRepository, err := ocitest.NewRepo(repo)
		Expect(err).ToNot(HaveOccurred())
		if needsAuth {
			mockRepository = ocitest.WithInsecureAuth(mockRepository, authuser, authpass)
		}
		mockReference = fmt.Sprintf("%s/%s:%s", mockImageRepository, mockImageName, mockImageVersion)
		_, err = ocitest.PushTestManifest(ctx, mockRepository, mockImageVersion)
		Expect(err).To(Succeed())
	}

	BeforeEach(func() {
		ctx = context.Background()
		cmd = NewPullCommand()
		cmd.SetContext(GinkgoT().Context())
		viper.SetConfigType("json")
		viper.Set("tmp-dir", arcctlTempDir)
	})

	AfterEach(func() {
		viper.Reset()
		Expect(os.RemoveAll(arcctlTempDir)).ToNot(HaveOccurred())
	})

	Context("when configuration is invalid", func() {
		It("should fail to pull the OCI artifact", func() {
			// setup config
			conf := &config.ArcctlConfig{}

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
			setupRegistry(true)
		})
		AfterEach(func() {
			mockRegistry.Close()
		})

		It("should pull the OCI artifact successfully with auth set", func() {
			// setup config
			conf := &config.ArcctlConfig{}
			conf.Type = config.AT_OCI
			conf.Src.Type = config.AT_OCI
			conf.Src.RemoteURL = mockRegistry.Listener.Addr().String()
			conf.Src.Auth = config.OCIAuth{
				Username: authuser,
				Password: authpass,
			}
			conf.Spec = config.OCISpec{
				Image: mockReference,
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

			// actually pull
			err = runPull(cmd, []string{})
			Expect(err).To(HaveOccurred())
		})
	})
})
