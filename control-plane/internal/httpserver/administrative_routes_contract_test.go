package httpserver

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	"waf/control-plane/apiroutes"
)

func TestAdministrativeRouteCatalogMatchesServerMux(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "server.go", nil, 0)
	if err != nil {
		t.Fatalf("parse server routes: %v", err)
	}
	actual := map[string]struct{}{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Handle" {
			return true
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		path, err := strconv.Unquote(literal.Value)
		if err == nil && len(path) >= len("/api/") && path[:len("/api/")] == "/api/" {
			actual[path] = struct{}{}
		}
		return true
	})
	expected := map[string]struct{}{}
	for _, path := range apiroutes.AdministrativePaths {
		expected[path] = struct{}{}
	}
	for path := range actual {
		if _, ok := expected[path]; !ok {
			t.Errorf("new administrative route %q is missing from apiroutes.AdministrativePaths", path)
		}
	}
	for path := range expected {
		if _, ok := actual[path]; !ok {
			t.Errorf("catalog route %q is not registered by the control-plane mux", path)
		}
	}
}
