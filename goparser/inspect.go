package goparser

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/packages"
)

type inspector struct{}

func NewInspector() *inspector {
	return &inspector{}
}

func (i *inspector) FindDeployablesAndFileSystems(pkgs []*packages.Package) [][]EmbeddedFsInfo {
	verbose := false
	deployables := [][]EmbeddedFsInfo{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				pos := pkg.Fset.Position(call.Pos())
				isIncludeFunc, err := IsIncludeFunc(call)
				if err != nil {
					if verbose {
						fmt.Printf("%v at %v", err, pos.Filename)
					}
					return true
				}
				if !isIncludeFunc {
					return true
				}

				deployables = append(deployables, GetEmbeddedFS(pkg, call))
				return true
			})
		}
	}
	return deployables
}
