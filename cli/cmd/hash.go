package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	hashCmd = &cobra.Command{
		Use:   "hash",
		Short: "Compute a suitable hash to use as schema suffix",
		RunE: func(cmd *cobra.Command, args []string) error {
			deployable, err := dep(false)
			if err != nil {
				return err
			}

			fmt.Println(deployable.SchemaSuffix)
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(hashCmd)
}
