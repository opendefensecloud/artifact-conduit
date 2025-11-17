// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opendefense.cloud/arc/pkg/workflow/config"
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

	if !viper.IsSet("tmp-dir") {
		return fmt.Errorf("tmp-dir is not set")
	}
	tmpDir := viper.GetString("tmp-dir")
	plainHTTP := viper.GetBool("plain-http")

	// Load configuration
	conf, err := config.LoadFromViper()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Only oci is supported for this command
	if err := conf.Validate(config.AT_OCI); err != nil {
		return err
	}

	// Get typed spec
	ociSpec := conf.GetOCISpec()
	srcReference := fmt.Sprintf("%s/%s", conf.Src.RemoteURL, ociSpec.Image)

	// Create source (remote OCI repository)
	repo, err := remote.NewRepository(srcReference)
	if err != nil {
		return fmt.Errorf("failed to get source repository: %w", err)
	}
	repo.PlainHTTP = plainHTTP

	// Set up authentication if provided
	if conf.Src.Auth != nil {
		// Get typed auth
		ociAuth := conf.Src.GetOCIAuth()
		repo.Client = &auth.Client{
			Client: retry.DefaultClient,
			Cache:  auth.NewCache(),
			Credential: auth.StaticCredential(repo.Reference.Registry, auth.Credential{
				Username: ociAuth.Username,
				Password: ociAuth.Password,
			}),
		}
	}

	// Create destination (local OCI layout)
	dst, err := oci.NewWithContext(ctx, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	// Copy artifact from source to destination
	fmt.Printf("Pulling artifact from %s\n", srcReference)
	desc, err := oras.Copy(ctx, repo, srcReference, dst, srcReference, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to copy artifact: %w", err)
	}

	fmt.Printf("Pulled artifact with digest: %s\n", desc.Digest.String())
	return nil
}
