package cmd

import (
	"errors"
	"fmt"
	"go/ast"
	"os"

	"github.com/spf13/cobra"
	"github.com/vippsas/sqlcode/goparser"
)

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

			//finder := goparser.NewInspector().FindDeployablesAndFileSystems
			finder := goparser.NewWalker().FindDeployablesAndFileSystems

			deployables := findDeployablesAndFileSystems(dir, finder)
			println("dump")
			for deployable, fss := range deployables {
				fmt.Printf("deployable %d %v\n", deployable)
				for j, fs := range fss {
					fmt.Printf("fileSystem %d\n", j)
					fmt.Printf("%s\n", fs.Package)
					fmt.Printf("%s\n", fs.Object)
					embeddedFiles, err := goparser.GetEmbbededFiles(fs.Object.Pkg().Path())
					if err != nil {
						continue
					}
					fmt.Printf("%v\n", embeddedFiles)
					fmt.Printf("\n")
				}
			}

			return nil
		},
	}
)

func findDeployablesAndFileSystems(dir string, finder goparser.Finder) map[ast.Node][]goparser.EmbeddedFsInfo {
	pkgs, err := goparser.GetPackages(dir)
	if err != nil {
		fmt.Printf("Error loading package: %v\n", err)
		return nil
	}

	info := finder(pkgs)
	return info
}

func init() {
	rootCmd.AddCommand(sqlFsCmd)
}
