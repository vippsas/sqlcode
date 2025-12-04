package cmd

import (
	"errors"
	"fmt"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/spf13/cobra"
	"github.com/vippsas/sqlcode"
)

var (
	buildCmd = &cobra.Command{
		Use:   "build schemasuffix",
		Short: "Dump the SQL that will be executed to populate the [code] schema to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				_ = cmd.Help()
				return errors.New("need to specify argument <schemasuffix>")
			}
			schemasuffix := args[0]

			d, err := dep(false)
			if err != nil {
				return err
			}

			preprocessed, err := sqlcode.Preprocess(d.CodeBase, schemasuffix, &mssql.Driver{})
			if err != nil {
				return err
			}
			for _, p := range preprocessed.Batches {
				fmt.Println(p.Lines)
				fmt.Println("===")
			}
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(buildCmd)
}
