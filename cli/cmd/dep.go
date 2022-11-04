package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/vippsas/sqlcode"
	"os"
)

func dep(partialParseResults bool) (d sqlcode.Deployable, err error) {
	d, err = sqlcode.Include(
		sqlcode.Options{
			IncludeTags:         tags,
			PartialParseResults: partialParseResults,
		},
		os.DirFS(directory),
	)
	return
}

var (
	depCmd = &cobra.Command{
		Use:   "dep",
		Short: "Scan the directory trees and report which files were discovered, their ordering and their dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				_ = cmd.Help()
				return errors.New("Too many arguments")
			}
			d, err := dep(true)
			if err != nil {
				fmt.Println("Error during parsing: " + err.Error())
				fmt.Println("Treat results below with caution.")
				fmt.Println()
				err = nil
			}
			if len(d.CodeBase.Creates) == 0 && len(d.CodeBase.Declares) == 0 {
				fmt.Println("No SQL code found in given paths")
			}
			if len(d.CodeBase.Errors) > 0 {
				fmt.Println("Errors:\n")
				for _, e := range d.CodeBase.Errors {
					fmt.Printf("%s:%d:%d: %s\n", e.Pos.File, e.Pos.Line, e.Pos.Line, e.Message)
				}
			}
			for _, c := range d.CodeBase.Creates {
				fmt.Println(c.QuotedName.String() + ":")
				if len(c.DependsOn) > 0 {
					fmt.Println("  Uses:")
					for _, u := range c.DependsOn {
						fmt.Println("    " + u.String())
					}
				}
				fmt.Println()
			}
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(depCmd)
}
