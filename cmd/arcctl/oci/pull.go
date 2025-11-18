// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
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
	httpClient := retry.DefaultClient

	// Load configuration
	if err := loadViperConfig(); err != nil {
		return err
	}

	// Get typed spec
	srcReference := conf.GetOCIReference(conf.Src.RemoteURL)

	// Create source (remote OCI repository)
	repo, err := remote.NewRepository(srcReference)
	if err != nil {
		return fmt.Errorf("failed to get source repository: %w", err)
	}
	repo.PlainHTTP = plainHTTP // allow plain http via args
	repo.Client = httpClient

	// allow insecure connection if configured via args
	if allowInsecureConnection {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// Set up authentication if provided
	if conf.Src.Auth != nil {
		ociAuth := conf.Src.GetOCIAuth()
		repo.Client = &auth.Client{
			Client: httpClient,
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
