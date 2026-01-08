package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	constantsCmd = &cobra.Command{
		Use:   "constants",
		Short: "Scan the directory trees and print out the constants declared",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				_ = cmd.Help()
				return errors.New("too many arguments")
			}
			d, err := dep(true)
			if err != nil {
				return err
			}
			if d.CodeBase.Empty() {
				fmt.Println("No SQL code found in given paths")
			}
			if d.CodeBase.HasErrors() {
				fmt.Println("Errors:")
				for _, e := range d.CodeBase.Errors() {
					fmt.Printf("%s:%d:%d: %s\n", e.Pos.File, e.Pos.Line, e.Pos.Line, e.Message)
				}
				return nil
			}
			fmt.Println("declare")
			for i, c := range d.CodeBase.Declares() {
				var prefix string
				if i == 0 {
					prefix = "    "
				} else {
					prefix = "  , "
				}
				fmt.Printf("%s%s %s = %s\n", prefix, c.VariableName, c.Datatype.String(), c.Literal.RawValue)
			}
			fmt.Println(";")
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(constantsCmd)
}
