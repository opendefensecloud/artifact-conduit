// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	clictx "ocm.software/ocm/api/cli"
	"ocm.software/ocm/cmds/ocm/app"
)

func newOCMCmd() *cobra.Command {
	c, err := app.NewCliCommandForArgs(clictx.DefaultContext(), os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
	return c
}
