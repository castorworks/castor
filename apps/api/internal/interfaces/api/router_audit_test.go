package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// auditExemptRoutes 列出不写审计日志的写操作路由及理由。新增写路由默认必须审计；
// 确实不改变业务数据或已在别处审计的，才在这里登记并写明原因。
var auditExemptRoutes = map[string]string{
	"POST /api/v1/auth/login":                        "audited by LoginService.recordLogin together with the login history",
	"POST /api/v1/account/identities/:provider/link": "only returns the provider's authorization URL; the link itself is audited in the OIDC callback",
	"POST /api/v1/auth/login/totp":                   "second sign-in step; audited by LoginService.RecordLogin together with the login history",
	"POST /api/v1/auth/refresh-token":                "rotates the caller's own token; no business data changes",
	"POST /api/v1/auth/code":                         "only sends a one-time code (rate limited); the flow that consumes it is audited",
	"POST /api/v1/account/contact/code":              "only sends a one-time code (rate limited); binding the contact is audited",
	"PUT /api/v1/account/notifications/:id/read":     "per-recipient read marker on the caller's own inbox",
	"PUT /api/v1/account/notifications/batch-read":   "per-recipient read marker on the caller's own inbox",
	"PUT /api/v1/account/notifications/read-all":     "per-recipient read marker on the caller's own inbox",
	"DELETE /api/v1/account/notifications/:id":       "hides a delivered notification from the caller's own inbox",
}

// auditCallNames 是写入审计日志的调用：handler 包统一经由 logAudit / logAuditAs（handler/audit.go），
// 中间件（登出）直接调用 AuditLogService.LogAsync。
var auditCallNames = map[string]bool{"LogAsync": true, "logAudit": true, "logAuditAs": true}

// TestMutatingRoutesWriteAuditLogs 守护"每个写操作都要留痕"：沿 router.go 找到每个
// POST/PUT/PATCH/DELETE 路由的处理函数，要求它（或它调用的同接收者方法、同包函数）
// 调用了审计写入。失败分支是否留痕由 TestHandlerAuditCallsRecordOutcome 守护。
func TestMutatingRoutesWriteAuditLogs(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate router test source")
	}
	dir := filepath.Dir(filename)
	routerFile, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, "router.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	handlerTypes := handlerGroupFieldTypes(routerFile)
	methods := parseMethods(t, filepath.Join(dir, "handler"), filepath.Join(dir, "middleware"))

	routes := collectHandlerRoutes(routerFile)
	if len(routes) == 0 {
		t.Fatal("no routes parsed from router.go")
	}
	seenExemptions := map[string]bool{}
	var missing []string
	for _, route := range routes {
		if route.method == "GET" {
			continue
		}
		key := route.method + " " + route.path
		if _, ok := auditExemptRoutes[key]; ok {
			seenExemptions[key] = true
			continue
		}
		typeName, method, ok := resolveHandler(route.handler, handlerTypes)
		if !ok {
			missing = append(missing, key+" (handler not resolvable: "+route.handler+")")
			continue
		}
		if !methods.audits(typeName, method, map[string]bool{}) {
			missing = append(missing, key+" -> "+typeName+"."+method)
		}
	}
	for key := range auditExemptRoutes {
		if !seenExemptions[key] {
			missing = append(missing, key+" (stale exemption: route no longer exists)")
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("mutating routes without an audit log write:\n  %s", strings.Join(missing, "\n  "))
	}
}

type handlerRoute struct {
	method  string
	path    string
	handler string // selector chain of the last argument, e.g. "r.admin.User.Put"
}

// collectHandlerRoutes 复用 router_rbac_test 的分组解析，额外记下每个路由的处理函数表达式。
func collectHandlerRoutes(file *ast.File) []handlerRoute {
	groups := map[string]string{"engine": ""}
	var routes []handlerRoute
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
				if baseOK && literalOK {
					groups[name.Name] = groups[base.Name] + suffix
				}
			case *ast.ExprStmt:
				call, ok := node.X.(*ast.CallExpr)
				if !ok || len(call.Args) < 2 {
					continue
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !isHTTPMethod(selector.Sel.Name) {
					continue
				}
				group, ok := selector.X.(*ast.Ident)
				if !ok {
					continue
				}
				suffix, ok := stringLiteral(call.Args[0])
				if !ok {
					continue
				}
				routes = append(routes, handlerRoute{
					method:  selector.Sel.Name,
					path:    groups[group.Name] + suffix,
					handler: selectorChain(call.Args[len(call.Args)-1]),
				})
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
	return routes
}

func selectorChain(expression ast.Expr) string {
	switch node := expression.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		return selectorChain(node.X) + "." + node.Sel.Name
	default:
		return ""
	}
}

// handlerGroupFieldTypes 把 AdminHandlers/UserHandlers 的字段映射到处理器类型名，
// 例如 "admin.User" -> "AdminUserHandler"。
func handlerGroupFieldTypes(file *ast.File) map[string]string {
	result := map[string]string{}
	prefixes := map[string]string{"AdminHandlers": "admin", "UserHandlers": "user"}
	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.TypeSpec)
		if !ok {
			return true
		}
		prefix, ok := prefixes[spec.Name.Name]
		structType, isStruct := spec.Type.(*ast.StructType)
		if !ok || !isStruct {
			return true
		}
		for _, field := range structType.Fields.List {
			star, ok := field.Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			typeSelector, ok := star.X.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			for _, name := range field.Names {
				result[prefix+"."+name.Name] = typeSelector.Sel.Name
			}
		}
		return false
	})
	return result
}

func resolveHandler(chain string, handlerTypes map[string]string) (typeName, method string, ok bool) {
	parts := strings.Split(chain, ".")
	switch {
	case len(parts) == 4 && parts[0] == "r" && (parts[1] == "admin" || parts[1] == "user"):
		typeName, ok = handlerTypes[parts[1]+"."+parts[2]]
		return typeName, parts[3], ok
	case len(parts) == 3 && parts[0] == "r" && parts[1] == "jwtMiddleware":
		return "JwtMiddleware", parts[2], true
	default:
		return "", "", false
	}
}

// methodIndex 按"接收者类型.方法名"和"包函数名"索引处理器与中间件包的函数声明。
type methodIndex map[string]*ast.FuncDecl

func parseMethods(t *testing.T, dirs ...string) methodIndex {
	t.Helper()
	index := methodIndex{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, name), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Body == nil {
					continue
				}
				index[receiverType(function)+"."+function.Name.Name] = function
			}
		}
	}
	return index
}

func receiverName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 || len(function.Recv.List[0].Names) == 0 {
		return ""
	}
	return function.Recv.List[0].Names[0].Name
}

func receiverType(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return ""
	}
	expression := function.Recv.List[0].Type
	if star, ok := expression.(*ast.StarExpr); ok {
		expression = star.X
	}
	if ident, ok := expression.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// audits 判断函数体（含其中的闭包、同接收者方法与同包函数调用）是否写了审计日志。
func (index methodIndex) audits(typeName, method string, visited map[string]bool) bool {
	key := typeName + "." + method
	if visited[key] {
		return false
	}
	visited[key] = true
	function, ok := index[key]
	if !ok {
		return false
	}
	receiver := receiverName(function)
	found := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if auditCallNames[fun.Sel.Name] {
				found = true
				return false
			}
			// 只追踪经由接收者本身的调用（h.save(...)），h.Service.Put(...) 不是本类型的方法。
			if base, ok := fun.X.(*ast.Ident); ok && receiver != "" && base.Name == receiver && index.audits(typeName, fun.Sel.Name, visited) {
				found = true
				return false
			}
		case *ast.Ident:
			if auditCallNames[fun.Name] || index.audits("", fun.Name, visited) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// TestHandlerAuditCallsRecordOutcome 守护"失败也要留痕"：handler 包只能经由 logAudit / logAuditAs
// 写审计日志，且结果参数必须是操作返回的 err。传字面量 nil 就是只在成功分支记录——
// 越权尝试、冲突和失败会无迹可查；直接调用 LogAsync 会绕过失败原因的脱敏。
func TestHandlerAuditCallsRecordOutcome(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate router test source")
	}
	dir := filepath.Join(filepath.Dir(filename), "handler")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var problems []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.SelectorExpr:
				if fun.Sel.Name == "LogAsync" && name != "audit.go" {
					problems = append(problems, fset.Position(call.Pos()).String()+": direct LogAsync call; use logAudit / logAuditAs")
				}
			case *ast.Ident:
				if fun.Name != "logAudit" && fun.Name != "logAuditAs" {
					return true
				}
				last, isIdent := call.Args[len(call.Args)-1].(*ast.Ident)
				if isIdent && last.Name == "nil" {
					problems = append(problems, fset.Position(call.Pos()).String()+": audit outcome is a literal nil; pass the operation's err so failures are recorded too")
				}
			}
			return true
		})
	}
	if len(problems) != 0 {
		t.Fatalf("handler audit calls must record the outcome:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestAuditLogTypesInUse 守护审计类型与操作一一对应：每个 AuditLogType 常量都要有代码在用。
// 没人用的类型通常意味着某处操作被记成了别的类型（例如菜单改动记成"新增权限"）。
func TestAuditLogTypesInUse(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate router test source")
	}
	internal := filepath.Join(filepath.Dir(filename), "..", "..")
	entityPath := filepath.Join(internal, "domain", "audit_log", "entity.go")
	entity, err := parser.ParseFile(token.NewFileSet(), entityPath, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var constants []string
	ast.Inspect(entity, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for _, name := range spec.Names {
			if strings.HasPrefix(name.Name, "AuditLogType") {
				constants = append(constants, name.Name)
			}
		}
		return false
	})
	if len(constants) == 0 {
		t.Fatal("no AuditLogType constants parsed")
	}

	used := map[string]bool{}
	err = filepath.WalkDir(internal, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || path == entityPath {
			return err
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, constant := range constants {
			if strings.Contains(string(source), "audit_log."+constant) {
				used[constant] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var unused []string
	for _, constant := range constants {
		if !used[constant] {
			unused = append(unused, constant)
		}
	}
	if len(unused) != 0 {
		t.Fatalf("audit log types that no code records (misfiled operations or dead types):\n  %s", strings.Join(unused, "\n  "))
	}
}
