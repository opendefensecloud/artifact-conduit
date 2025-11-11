// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"
	orasRoot "oras.land/oras/cmd/oras/root"
)

func newOrasCmd() *cobra.Command {
	cmd := orasRoot.New()
	cmd.Short = "ORAS CLI (Artifact Registry As Storage)"
	cmd.Long = `ORAS CLI (Artifact Registry As Storage) is a CLI tool for working with OCI registries.

ORAS CLI allows you to push, pull, and manage artifacts in OCI-compliant registries,
enabling you to leverage container registries for storing and distributing various types of artifacts.`
	return cmd
}
