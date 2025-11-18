// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !release

package oci

import (
	"context"
	"fmt"

	"github.com/onsi/gomega"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	MockImageRepository = "arc"
	MockImageName       = "test"
	MockImageVersion    = "v0.0.0"
	AuthUser            = "user"
	AuthPass            = "pass"
)

type TestEnv struct {
	MockRegistry  *Registry
	MockRepoURL   string
	MockRepo      *remote.Repository
	MockReference string
}

// NewTestEnv creates a new TestEnv instance.
func NewTestEnv() *TestEnv {
	return &TestEnv{}
}

// Setup sets up the environment for testing.
func (env *TestEnv) Setup(ctx context.Context) {
	env.MockRegistry = NewRegistry()

	// create oras repo
	env.MockRepoURL = fmt.Sprintf("%s/%s/%s", env.MockRegistry.Listener.Addr().String(), MockImageRepository, MockImageName)
	mockRepository, err := remote.NewRepository(env.MockRepoURL)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	mockRepository.PlainHTTP = true

	// push test image
	env.MockReference = fmt.Sprintf("%s/%s:%s", MockImageRepository, MockImageName, MockImageVersion)
	_, err = PushTestManifest(ctx, mockRepository, MockImageVersion)
	gomega.Expect(err).To(gomega.Succeed())
}

// EnableAuth enables authentication for the registry of test environment.
func (env *TestEnv) EnableAuth() {
	env.MockRegistry = env.MockRegistry.WithAuth(AuthUser, AuthPass)
}

// Shutdown shuts down the test environment.
func (env *TestEnv) Shutdown(_ context.Context) {
	env.MockRegistry.Close()
}
