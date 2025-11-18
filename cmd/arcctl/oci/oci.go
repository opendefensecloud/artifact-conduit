// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opendefense.cloud/arc/pkg/workflow/config"
)

var (
	tmpDir                  string
	plainHTTP               bool
	allowInsecureConnection bool
	conf                    *config.WorkflowConfig
)

func NewOCICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "oci",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Short:                 "Subcommand to manage OCI artifacts",
	}

	pflags := cmd.PersistentFlags()
	_ = pflags.Bool("plain-http", false, "allow insecure connections to registry without SSL check")
	_ = pflags.Bool("insecure", false, "allow connections to SSL registry without certs")
	cmd.AddCommand(NewPullCommand())
	cmd.AddCommand(NewPushCommand())

	return cmd
}

func loadViperConfig() error {
	// Validate flags/config/env
	if !viper.IsSet("tmp-dir") {
		return fmt.Errorf("tmp-dir is not set")
	}
	tmpDir = viper.GetString("tmp-dir")
	plainHTTP = viper.GetBool("plain-http")
	allowInsecureConnection = viper.GetBool("insecure")

	// Load workflow config
	wc, err := config.LoadFromViper()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Only oci is supported for this subcommand
	if err := wc.Validate(config.AT_OCI); err != nil {
		return err
	}
	conf = wc

	return nil
}
