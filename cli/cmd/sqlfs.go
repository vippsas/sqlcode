package cmd

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
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

			deployables := findDeployablesAndFileSystems(dir)
			for _, deployable := range deployables {
				for _, fileSystem := range deployable {
					fmt.Printf("%s\n", fileSystem)
				}
			}

			return nil
		},
	}
)

type info struct {
	p *packages.Package
	o types.Object
}

func findDeployablesAndFileSystems(dir string) [][]info {
	verbose := false

	// Load the package containing your .go file
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  dir, // Or set to the directory containing your file
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Printf("Error loading package: %v\n", err)
		return nil
	}

	deployables := [][]info{}

	// Iterate through syntax trees
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				//fmt.Printf("Found a call at %v\n", pkg.Fset.Position(call.Pos()))
				var funcName string
				pos := pkg.Fset.Position(call.Pos())
				switch fun := call.Fun.(type) {
				case *ast.Ident:
					funcName = fun.Name
				case *ast.SelectorExpr:
					funcName = fmt.Sprintf("%s.%s", exprToString(fun.X), fun.Sel.Name)
				default:
					if verbose {
						fmt.Printf("Unhandled call type %v at %v\n", reflect.TypeOf(fun), pos)
					}
				}
				//fmt.Printf("func %v\n", funcName)

				if !strings.Contains(funcName, "MustInclude") && !strings.Contains(funcName, "Include") {
					return true
				}

				fmt.Printf("Found call %v at %v\n", funcName, pos)
				fileSystems := []info{}

				for _, arg := range call.Args {
					// Get type and object info
					tv := pkg.TypesInfo.Types[arg]
					obj := pkg.TypesInfo.Uses[identOf(arg)]
					if tv.Type.String() != "embed.FS" {
						continue
					}

					fmt.Printf("Arg: %s, Type: %s\n", exprToString(arg), tv.Type)
					if obj != nil {
						pos := pkg.Fset.Position(obj.Pos())
						fmt.Printf("Declared at: %s (%s)\n", obj.Name(), pos)
						fileSystems = append(fileSystems, info{
							p: pkg,
							o: obj,
						})
					}
				}
				deployables = append(deployables, fileSystems)
				return true
			})
		}
	}
	return deployables
}

// Helper: get identifier if available
func identOf(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return e.Sel
	default:
		return nil
	}
}

// Helper: format expression (simplified)
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.BasicLit:
		return e.Value
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", exprToString(e.X), e.Sel.Name)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func init() {
	rootCmd.AddCommand(sqlFsCmd)
}
