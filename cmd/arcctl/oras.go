package main

import (
	"github.com/spf13/cobra"
	orasRoot "oras.land/oras/cmd/oras/root"
)

func newOrasCmd() *cobra.Command {
	cmd := orasRoot.New()
	return cmd
}
