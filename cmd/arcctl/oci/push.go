// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func NewPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push an OCI artifact",
		Long:  `Push OCI artifact using ORAS`,
		RunE:  runPush,
	}
	return cmd
}

func runPush(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Load configuration
	if err := loadViperConfig(); err != nil {
		return err
	}

	// Get typed spec
	dstReference := conf.GetOCIReference(conf.Dst.RemoteURL)

	// Create destination (remote OCI repository)
	repo, err := remote.NewRepository(dstReference)
	if err != nil {
		return fmt.Errorf("failed to get destination repository: %w", err)
	}
	repo.PlainHTTP = plainHTTP

	httpClient := retry.DefaultClient
	repo.Client = httpClient

	// allow insecure connection
	if insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// Set up authentication if provided
	if viper.IsSet("destination.auth") {
		srcUser := viper.GetString("destination.auth.username")
		srcPassword := viper.GetString("destination.auth.password")

		repo.Client = &auth.Client{
			Client: httpClient,
			Cache:  auth.NewCache(),
			Credential: auth.StaticCredential(repo.Reference.Registry, auth.Credential{
				Username: srcUser,
				Password: srcPassword,
			}),
		}
	}

	// Create source (local OCI layout)
	src, err := oci.NewWithContext(ctx, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}

	// Copy artifact from source to destination
	fmt.Printf("Pushing artifact to %s\n", dstReference)
	if err := src.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			desc, err := oras.Copy(ctx, src, tag, repo, dstReference, oras.DefaultCopyOptions)
			if err != nil {
				return fmt.Errorf("failed to copy artifact: %w", err)
			}

			fmt.Printf("Pushed artifact with digest: %s\n", desc.Digest.String())
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to list tags in source: %w", err)
	}
	return nil
}
