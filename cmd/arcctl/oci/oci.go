package oci

import (
	"github.com/spf13/cobra"
)

func NewOCICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "oci",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Short:                 "Subcommand to manage OCI artifacts",
	}

	cmd.AddCommand(NewPullCommand())
	cmd.AddCommand(NewPushCommand())

	return cmd
}
