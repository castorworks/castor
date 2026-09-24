package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/castorworks/castor/internal/interfaces/api/openapi"
)

var updateOpenAPI = flag.Bool("update-openapi", false, "rewrite apps/api/openapi.json from the route catalog")

// TestOpenAPICatalogCoversEveryRoute 每个注册的路由都在文档目录里，目录里没有已删除的路由，
// 且登记的鉴权级别与路由组实际挂的中间件一致。
func TestOpenAPICatalogCoversEveryRoute(t *testing.T) {
	declared := map[string]routeAuth{}
	for _, route := range parseRoutes(t) {
		declared[route.method+" "+route.path] = route.auth
	}
	documented := map[string]openapi.Auth{}
	for _, op := range apiOperations {
		documented[op.Method+" "+op.Path] = op.Auth
	}
	for key := range declared {
		if _, ok := documented[key]; !ok {
			t.Errorf("route %s is not in apiOperations (openapi_catalog.go)", key)
		}
	}
	for key, auth := range documented {
		level, ok := declared[key]
		if !ok {
			t.Errorf("apiOperations documents %s, which router.go does not register", key)
			continue
		}
		want := map[routeAuth]openapi.Auth{authPublic: openapi.Public, authSession: openapi.Session, authAdmin: openapi.Admin}[level]
		if auth != want {
			t.Errorf("%s is documented as %s but the router makes it %s", key, auth, want)
		}
	}
}

// TestOpenAPIErrorCodesAreDescribed response/errors.go 里每个业务错误码都要在文档里有说明。
func TestOpenAPIErrorCodesAreDescribed(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(filename), "response", "errors.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range spec.Names {
			if !strings.HasPrefix(name.Name, "ErrCode") || name.Name == "ErrCodeSuccess" || i >= len(spec.Values) {
				continue
			}
			lit, ok := spec.Values[i].(*ast.BasicLit)
			if !ok {
				continue
			}
			count++
			var code int
			if err := json.Unmarshal([]byte(lit.Value), &code); err != nil {
				t.Fatal(err)
			}
			if _, ok := errorCodeDescriptions[code]; !ok {
				t.Errorf("%s (%d) has no description in errorCodeDescriptions", name.Name, code)
			}
		}
		return true
	})
	if count == 0 {
		t.Fatal("no error codes were parsed from response/errors.go")
	}
}

// TestOpenAPISpecIsUpToDate 提交在仓库里的 apps/api/openapi.json 必须与目录生成的一致。
// 改了路由、DTO 或目录后运行：go test ./internal/interfaces/api -run TestOpenAPISpecIsUpToDate -update-openapi
func TestOpenAPISpecIsUpToDate(t *testing.T) {
	doc, err := BuildOpenAPI()
	if err != nil {
		t.Fatalf("BuildOpenAPI() error = %v", err)
	}
	generated, err := doc.JSON()
	if err != nil {
		t.Fatal(err)
	}
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(filename), "..", "..", "..", "openapi.json")
	if *updateOpenAPI {
		if err := os.WriteFile(path, generated, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (generate it with -update-openapi)", path, err)
	}
	if !bytes.Equal(bytes.ReplaceAll(committed, []byte("\r\n"), []byte("\n")), generated) {
		t.Fatal("apps/api/openapi.json is stale; run: go test ./internal/interfaces/api -run TestOpenAPISpecIsUpToDate -update-openapi")
	}
}
