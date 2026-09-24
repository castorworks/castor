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

// routeAuth 路由需要的身份：由所在路由组挂的中间件决定
type routeAuth int

const (
	authPublic routeAuth = iota
	authSession
	authAdmin
)

type declaredRoute struct {
	group  string
	method string
	path   string
	auth   routeAuth
}

// parseRoutes 从 router.go 的 With 函数读出全部路由（完整路径模板、方法与鉴权级别）。
// 读源码而不是构造 Router：路由目录类测试不需要装配任何依赖。
func parseRoutes(t *testing.T) []declaredRoute {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate router test source")
	}
	file, err := parser.ParseFile(token.NewFileSet(), strings.TrimSuffix(filename, "routes_parse_test.go")+"router.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	groups := map[string]string{"engine": ""}
	parents := make(map[string]string)
	levels := make(map[string]routeAuth)
	var routes []declaredRoute

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
				if !nameOK || !callOK {
					continue
				}
				selector, selectorOK := call.Fun.(*ast.SelectorExpr)
				if !selectorOK || selector.Sel.Name != "Group" || len(call.Args) == 0 {
					continue
				}
				base, baseOK := selector.X.(*ast.Ident)
				suffix, literalOK := stringLiteral(call.Args[0])
				if !baseOK || !literalOK {
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
						ast.Inspect(argument, func(n ast.Node) bool {
							field, ok := n.(*ast.SelectorExpr)
							if !ok {
								return true
							}
							switch {
							case field.Sel.Name == "adminRoleMiddleware":
								levels[group.Name] = authAdmin
							case field.Sel.Name == "MiddlewareFunc" && levels[group.Name] < authSession:
								levels[group.Name] = authSession
							}
							return true
						})
					}
					continue
				}
				if !isHTTPMethod(selector.Sel.Name) || len(call.Args) == 0 {
					continue
				}
				if suffix, ok := stringLiteral(call.Args[0]); ok {
					routes = append(routes, declaredRoute{group: group.Name, method: selector.Sel.Name, path: groups[group.Name] + suffix})
				}
			case *ast.BlockStmt:
				scanBlock(node)
			}
		}
	}
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "With" {
			scanBlock(function.Body)
		}
	}
	if len(routes) == 0 {
		t.Fatal("no routes were parsed from router.go; the route guardrails would silently pass")
	}

	levelOf := func(group string) routeAuth {
		level := authPublic
		for group != "" {
			level = max(level, levels[group])
			group = parents[group]
		}
		return level
	}
	for i := range routes {
		routes[i].auth = levelOf(routes[i].group)
	}
	return routes
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
