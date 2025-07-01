package cmd

import (
	"errors"
	"fmt"
	"go/ast"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
	"github.com/vippsas/sqlcode"
	"github.com/vippsas/sqlcode/go/mapfs"
	"github.com/vippsas/sqlcode/go/parser"
	"golang.org/x/tools/go/packages"
)

// TODO: Handle include tags

var (
	sqlFsCmd = &cobra.Command{
		Use:   "sqlfs",
		Short: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 && len(args) != 1 {
				_ = cmd.Help()
				return errors.New("Too many arguments")
			}

			dir, err := os.Getwd()
			if err != nil {
				return err
			}

			if len(args) != 0 {
				dir = args[1]
			}

			deployables := func(
				dir string,
				finder func([]*packages.Package) map[ast.Node][]parser.EmbeddedFsInfo,
			) map[ast.Node][]parser.EmbeddedFsInfo {
				pkgs, err := parser.GetPackages(dir)
				if err != nil {
					fmt.Printf("Error loading package: %v\n", err)
					return nil
				}
				return finder(pkgs)
			}(dir, parser.NewWalker().FindDeployablesAndFileSystems)

			for deployable, fss := range deployables {
				embeddedFiles := mapfs.MapFS{}

				fmt.Printf("deployable %d %v\n", deployable)
				for j, fs := range fss {
					fmt.Printf("fileSystem %d\n", j)
					fmt.Printf("%s\n", fs.Package)
					fmt.Printf("%s\n", fs.Object)
					efs, err := parser.GetEmbbededFiles(fs.Object.Pkg().Path())
					if err != nil {
						continue
					}
					fmt.Printf("%v\n", efs)
					fmt.Printf("\n")
					for _, ef := range efs {
						embeddedFiles.Add(ef)
					}
				}
				VerifyEmbeddedFiles(embeddedFiles)
			}

			return nil
		},
	}
)

func VerifyEmbeddedFiles(files fs.FS) {
	// TODO(dsf): tags
	d, err := sqlcode.Include(
		sqlcode.Options{
			IncludeTags:         tags,
			PartialParseResults: true,
		},
		files,
	)
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
		fmt.Println("Errors:")
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
}

func init() {
	rootCmd.AddCommand(sqlFsCmd)
}
