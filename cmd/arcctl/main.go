// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jastBytes/sprint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.opencode.de/bwi/ace/artifact-conduit/cmd/arcctl/oci"
)

var (
	// cfgFile holds the path to the configuration file
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               "arcctl [command] [flags]",
	Short:             "CLI of ARC",
	Long:              `arcctl is the command line interface for Artefact Conduit (ARC).`,
	DisableAutoGenTag: true,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Initialize configuration and flags
func init() {
	cobra.OnInitialize(initConfig)

	// Setup global flags
	fl := rootCmd.PersistentFlags()
	fl.AddGoFlagSet(flag.CommandLine)
	fl.StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.config/arc/config.yaml)")
	_ = fl.StringP("tmp-dir", "t", "/tmp/arcctl", "Path to temporary directory")

	// Bind flags to viper
	sprint.PanicOnError(viper.BindPFlags(rootCmd.PersistentFlags()))

	// Add subcommands
	// rootCmd.AddCommand(newOrasCmd())
	// rootCmd.AddCommand(newOCMCmd())
	rootCmd.AddCommand(oci.NewOCICommand())
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name "config" (without extension).
		viper.AddConfigPath(fmt.Sprintf("%s/.config/arc", home))
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
