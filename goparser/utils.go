package goparser

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

type EmbeddedFsInfo struct {
	Package *packages.Package
	Object  types.Object
}

// type Finder func([]*packages.Package) [][]EmbeddedFsInfo
type Finder func([]*packages.Package) map[ast.Node][]EmbeddedFsInfo

func GetEmbbededFiles(pkgPath string) ([]string, error) {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax | packages.NeedEmbedFiles,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, err
	}

	println("here")
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected one package, %d found", len(pkgs))
	}
	pkg := pkgs[0]
	fmt.Printf("%#v\n", pkg.Name)

	return pkg.EmbedFiles, nil
}

/*
func GetPositionInPackage(obj types.Object) error {
	fset := pkg.Fset
	declPos := fset.Position(obj.Pos())
	fmt.Println("Declared at", declPos)

		// Now, to find the AST node at that position:
		found := false
		for _, file := range pkg.Syntax {
			if fset.Position(file.Pos()).Filename != declPos.Filename {
				continue
			}

			ast.Inspect(file, func(n ast.Node) bool {
				if n == nil {
					return false
				}
				if n.Pos() == obj.Pos() {
					fmt.Printf("Found declaration AST node: %T\n", n)
					found = true
					return false
				}
				return true
			})
		}

		if !found {
			fmt.Println("Declaration AST node not found — maybe external package?")
		}
	return nil
}
*/

func GetPackages(dir string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax | packages.NeedEmbedFiles,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}
	return pkgs, nil
}

var UnhandledCallType = errors.New("unhandled call type")

func IsIncludeFunc(call *ast.CallExpr) (bool, error) {
	var funcName string
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		funcName = fun.Name
	case *ast.SelectorExpr:
		funcName = fmt.Sprintf("%s.%s", exprToString(fun.X), fun.Sel.Name)
	default:
		return false, UnhandledCallType
	}
	if !strings.Contains(funcName, "MustInclude") && !strings.Contains(funcName, "Include") {
		return false, nil
	}
	return true, nil
}

func GetEmbeddedFS(pkg *packages.Package, call *ast.CallExpr) []EmbeddedFsInfo {
	embeddedFS := []EmbeddedFsInfo{}
	for _, arg := range call.Args {
		// Get type and object info
		tv := pkg.TypesInfo.Types[arg]
		obj := pkg.TypesInfo.Uses[identOf(arg)]
		if tv.Type.String() != "embed.FS" {
			continue
		}

		if obj != nil {
			embeddedFS = append(embeddedFS, EmbeddedFsInfo{
				Package: pkg,
				Object:  obj,
			})
		}
	}
	return embeddedFS
}

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
