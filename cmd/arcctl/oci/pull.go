// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func NewPullCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull an OCI artifact",
		Long:  `Pull OCI artifact using ORAS`,
		RunE:  runPull,
	}
	return cmd
}

func runPull(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Retrieve configuration values
	if !viper.IsSet("source.reference") {
		return fmt.Errorf("source.reference is not set")
	}
	srcReference := viper.GetString("source.reference")

	if !viper.IsSet("tmp-dir") {
		return fmt.Errorf("tmp-dir is not set")
	}
	tmpDir := viper.GetString("tmp-dir")

	// Create source (remote OCI repository)
	repo, err := remote.NewRepository(srcReference)
	if err != nil {
		return fmt.Errorf("failed to get source repository: %w", err)
	}

	fmt.Printf("Pulling artifact from %s\n", srcReference)

	// Set up authentication if provided
	if viper.IsSet("source.auth") {
		srcUser := viper.GetString("source.auth.username")
		srcPassword := viper.GetString("source.auth.password")

		repo.Client = &auth.Client{
			Client: retry.DefaultClient,
			Cache:  auth.NewCache(),
			Credential: auth.StaticCredential(repo.Reference.Registry, auth.Credential{
				Username: srcUser,
				Password: srcPassword,
			}),
		}
	}

	// Create destination (local OCI layout)

	dst, err := oci.NewWithContext(ctx, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	// Copy artifact from source to destination
	desc, err := oras.Copy(ctx, repo, srcReference, dst, srcReference, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to copy artifact: %w", err)
	}

	fmt.Printf("Pulled artifact with digest: %s\n", desc.Digest.String())
	return nil
}
