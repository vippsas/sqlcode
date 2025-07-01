package goparser

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/packages"
)

type walker struct{}

func NewWalker() *walker {
	return &walker{}
}

func (v *walker) FindDeployablesAndFileSystems(pkgs []*packages.Package) map[ast.Node][]EmbeddedFsInfo {
	deployables := make(map[ast.Node][]EmbeddedFsInfo)
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			visitor := &CallVisitor{
				pkg:         pkg,
				parentMap:   make(map[ast.Node]ast.Node),
				embeddedFSs: deployables,
			}
			ast.Walk(visitor, file)
		}
	}
	return deployables
}

type CallVisitor struct {
	pkg         *packages.Package
	parentMap   map[ast.Node]ast.Node
	parent      ast.Node
	embeddedFSs map[ast.Node][]EmbeddedFsInfo
}

func (v *CallVisitor) Visit(n ast.Node) ast.Visitor {
	verbose := false
	if n == nil {
		return nil // End of this branch
	}

	if v.parent != nil {
		v.parentMap[n] = v.parent
	}

	if verbose {
		fmt.Printf("Visiting node %v at %v\n", n, v.pkg.Fset.Position(n.Pos()))
	}

	if call, ok := n.(*ast.CallExpr); ok {
		if verbose {
			fmt.Printf("Visiting call %v\n", call)
		}
		if isIncludeFunc, err := IsIncludeFunc(call); err == nil && isIncludeFunc {
			if verbose {
				fmt.Printf("Is Include\n")
			}
			parent := v.parentMap[call]
			v.embeddedFSs[parent] = GetEmbeddedFS(v.pkg, call)
		}
	}

	next := &CallVisitor{
		pkg:         v.pkg,
		parentMap:   v.parentMap,
		parent:      n,
		embeddedFSs: v.embeddedFSs,
	}

	return next
}

func (v *CallVisitor) GetParentOfCall(call *ast.CallExpr) {
	fmt.Printf("Call to: %s at %v\n", exprToString(call.Fun), call.Pos())

	parent := v.parentMap[call]
	switch p := parent.(type) {
	case *ast.AssignStmt:
		fmt.Printf("  Assigned in assignment to: ")
		for _, lhs := range p.Lhs {
			fmt.Printf("%s ", exprToString(lhs))
		}
		fmt.Println()
	case *ast.ValueSpec:
		fmt.Printf("  Assigned in var declaration to: ")
		for _, name := range p.Names {
			fmt.Printf("%s ", name.Name)
		}
		fmt.Println()
	default:
		fmt.Println("  Return value not assigned (used directly or discarded)")
	}
}
