package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"testing"
)

func TestWalker(t *testing.T) {
	src := `
package main
func main() {
	x := foo()
	bar(foo())
}
func foo() int { return 42 }
func bar(int) {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", src, 0)
	if err != nil {
		log.Fatal(err)
	}

	visitor := &CallVisitor{
		parentMap: make(map[ast.Node]ast.Node),
	}
	ast.Walk(visitor, file)

	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			fmt.Printf("Call: %T at %v\n", call.Fun, fset.Position(call.Pos()))
			parent := visitor.parentMap[call]
			fmt.Printf("  Parent: %T\n", parent)
		}
		return true
	})
}
