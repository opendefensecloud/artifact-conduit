// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !release

package oci

import (
	"context"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	PlatformOSLinux   = "linux"
	PlatformOSMulti   = "multi"
	PlatformArchARM64 = "arm64"
	PlatformArchAMD64 = "amd64"
)

func PushTestManifest(ctx context.Context, repository, version string) (*ocispec.Descriptor, error) {
	repo, err := remote.NewRepository(repository)
	if err != nil {
		return nil, err
	}
	repo.PlainHTTP = true

	// push a layer to the repository
	layer := []byte("example manifest layer")
	layerDescriptor, err := oras.PushBytes(ctx, repo, ocispec.MediaTypeImageLayer, layer)
	if err != nil {
		return nil, err
	}

	// push a manifest to the repository with the tag "quickstart"
	packOpts := oras.PackManifestOptions{
		Layers: []ocispec.Descriptor{layerDescriptor},
	}
	artifactType := "application/vnd.example+type"
	desc, err := oras.PackManifest(ctx, repo, oras.PackManifestVersion1_1, artifactType, packOpts)
	if err != nil {
		return nil, err
	}

	err = repo.Tag(ctx, desc, version)
	if err != nil {
		return nil, err
	}

	return &desc, nil
}
