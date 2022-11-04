package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:          "sqlcode",
		Short:        "sqlcode",
		SilenceUsage: true,
		Long:         `CLI tool for migrating stored procedures/functions to Microsoft SQL, in an opinionated way. See README.md.`,
	}

	directory string
	tags      []string
)

// Execute executes the root command.
func Execute() error {
	rootCmd.PersistentFlags().StringVarP(&directory, "directory", "d", ".", "path to directory and subtree which will be scanned for *.sql-files")
	rootCmd.PersistentFlags().StringSliceVarP(&tags, "tags", "t", nil, "include tags; affects files that are included through the include-if pragma")
	return rootCmd.Execute()
}

func init() {
}
