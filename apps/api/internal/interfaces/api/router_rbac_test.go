package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/castorworks/castor/internal/domain/permission"
)

type declaredRoute struct {
	group  string
	method string
	path   string
}

func TestRBACResourcesCoverProtectedRoutes(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate router test source")
	}
	file, err := parser.ParseFile(token.NewFileSet(), strings.TrimSuffix(filename, "router_rbac_test.go")+"router.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	groups := map[string]string{"engine": ""}
	parents := make(map[string]string)
	protected := make(map[string]bool)
	routes := make([]declaredRoute, 0)

	var scanBlock func(*ast.BlockStmt)
	scanBlock = func(block *ast.BlockStmt) {
		for _, statement := range block.List {
			switch node := statement.(type) {
			case *ast.AssignStmt:
				if len(node.Lhs) != 1 || len(node.Rhs) != 1 {
					continue
				}
				name, nameOK := node.Lhs[0].(*ast.Ident)
				call, callOK := node.Rhs[0].(*ast.CallExpr)
				selector, selectorOK := call.Fun.(*ast.SelectorExpr)
				base, baseOK := selector.X.(*ast.Ident)
				if !nameOK || !callOK || !selectorOK || !baseOK || selector.Sel.Name != "Group" || len(call.Args) == 0 {
					continue
				}
				suffix, ok := stringLiteral(call.Args[0])
				if !ok {
					continue
				}
				groups[name.Name] = groups[base.Name] + suffix
				parents[name.Name] = base.Name
			case *ast.ExprStmt:
				call, ok := node.X.(*ast.CallExpr)
				if !ok {
					continue
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				group, ok := selector.X.(*ast.Ident)
				if !ok {
					continue
				}
				if selector.Sel.Name == "Use" {
					for _, argument := range call.Args {
						ast.Inspect(argument, func(node ast.Node) bool {
							if field, ok := node.(*ast.SelectorExpr); ok && field.Sel.Name == "adminRoleMiddleware" {
								protected[group.Name] = true
							}
							return true
						})
					}
					continue
				}
				if !isHTTPMethod(selector.Sel.Name) || len(call.Args) == 0 {
					continue
				}
				suffix, ok := stringLiteral(call.Args[0])
				if ok {
					routes = append(routes, declaredRoute{group: group.Name, method: selector.Sel.Name, path: groups[group.Name] + suffix})
				}
			case *ast.BlockStmt:
				scanBlock(node)
			}
		}
	}

	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "With" {
			scanBlock(function.Body)
		}
	}

	isProtected := func(group string) bool {
		for group != "" {
			if protected[group] {
				return true
			}
			group = parents[group]
		}
		return false
	}
	declared := make(map[string]struct{})
	for _, route := range routes {
		if isProtected(route.group) {
			declared[route.path+"\x00"+route.method] = struct{}{}
		}
	}

	configured := make(map[string]struct{})
	for _, resource := range permission.DefaultResources {
		for _, action := range resource.Actions {
			configured[resource.Path+"\x00"+action] = struct{}{}
		}
	}
	missing, stale := setDifference(declared, configured), setDifference(configured, declared)
	if len(missing) != 0 || len(stale) != 0 {
		t.Fatalf("RBAC route catalog drifted; missing=%v stale=%v", missing, stale)
	}
}

func stringLiteral(expression ast.Expr) (string, bool) {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

func isHTTPMethod(method string) bool {
	return method == permission.ActionGET || method == permission.ActionPOST || method == permission.ActionPUT || method == permission.ActionPATCH || method == permission.ActionDELETE
}

func setDifference(left, right map[string]struct{}) []string {
	result := make([]string, 0)
	for item := range left {
		if _, ok := right[item]; !ok {
			result = append(result, strings.Replace(item, "\x00", " ", 1))
		}
	}
	sort.Strings(result)
	return result
}
