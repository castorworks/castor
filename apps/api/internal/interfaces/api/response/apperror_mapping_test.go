package response

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// bypassesHandleError 列出不经过 HandleError 的哨兵错误，以及它们各自的出口。
// 新增条目必须写清楚替代出口，否则就是漏了映射。
var bypassesHandleError = map[string]string{
	// 登录失败由 JWT 中间件的 Unauthorized 分支转成 401，不走 HandleError。
	"ErrAuthenticationFailed": "middleware/auth.go Unauthorized",
	"ErrRateLimitCheckFailed": "middleware/auth.go Unauthorized",
	// init-db 命令行路径，永远不会作为 HTTP 响应返回。
	"ErrDefaultAdminPasswordRequired": "cmd init-db",
}

// TestEveryAppErrorHasHTTPMapping 防止新模块的哨兵错误漏掉 HTTP 映射。
//
// HandleError 在 errorMappings 里逐条 errors.Is，全部未命中就落到 500。
// 也就是说，漏登记的业务错误不会报错、不会有编译失败，只会在运行时
// 把一个本该是 409/400 的结果伪装成"服务器内部错误"，前端也拿不到可展示的 message。
// 新增模块时在 apperror 声明哨兵，就必须同时在 handle_error.go 的
// errorMappings 里给出错误码与 i18n key。
func TestEveryAppErrorHasHTTPMapping(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	apiRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")

	declared := declaredAppErrors(t, filepath.Join(apiRoot, "internal", "application", "apperror"))
	if len(declared) == 0 {
		t.Fatal("no apperror sentinels were parsed; the guardrail would silently pass")
	}
	mapped := mappedAppErrors(t, filepath.Join(apiRoot, "internal", "interfaces", "api", "response", "handle_error.go"))

	var missing []string
	for _, name := range declared {
		if mapped[name] {
			continue
		}
		if _, exempt := bypassesHandleError[name]; exempt {
			continue
		}
		missing = append(missing, name)
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("以下 apperror 哨兵没有 HTTP 映射，会被 HandleError 当成 500：%v\n"+
			"在 handle_error.go 的 errorMappings 中补充，或在 bypassesHandleError 中说明它的替代出口", missing)
	}

	// 豁免名单本身也会过期：错误被删掉或后来补了映射，都该及时清理。
	declaredSet := make(map[string]bool, len(declared))
	for _, name := range declared {
		declaredSet[name] = true
	}
	var stale []string
	for name := range bypassesHandleError {
		if !declaredSet[name] {
			stale = append(stale, name+"（已不存在）")
		} else if mapped[name] {
			stale = append(stale, name+"（已有映射）")
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("bypassesHandleError 已过期，请移除：%v", stale)
	}
}

// declaredAppErrors 收集 apperror 包里所有 `ErrXxx = ...` 顶层声明。
func declaredAppErrors(t *testing.T, dir string) []string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		t.Fatalf("parse apperror package: %v", err)
	}
	var names []string
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.VAR {
					continue
				}
				for _, spec := range gen.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range value.Names {
						if strings.HasPrefix(name.Name, "Err") {
							names = append(names, name.Name)
						}
					}
				}
			}
		}
	}
	sort.Strings(names)
	return names
}

// mappedAppErrors 收集 handle_error.go 中被 errorMappings 引用的哨兵。
func mappedAppErrors(t *testing.T, path string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse handle_error.go: %v", err)
	}
	mapped := make(map[string]bool)
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "apperror" {
			return true
		}
		mapped[sel.Sel.Name] = true
		return true
	})
	return mapped
}
