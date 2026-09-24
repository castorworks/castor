package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const modulePrefix = "github.com/castorworks/castor/internal/"

// layerRules 描述 DDD 分层的依赖约束：key 为 internal 下的目录前缀，value 为该层禁止导入的包前缀。
var layerRules = map[string][]string{
	"domain/": {
		modulePrefix + "application",
		modulePrefix + "infrastructure",
		modulePrefix + "interfaces",
		"gorm.io/",
		"github.com/gin-gonic/",
		"github.com/appleboy/gin-jwt",
		"github.com/redis/",
	},
	"application/": {
		modulePrefix + "infrastructure",
		modulePrefix + "interfaces",
		"gorm.io/",
		"github.com/gin-gonic/",
		"github.com/appleboy/gin-jwt",
	},
	"infrastructure/persistence/": {
		modulePrefix + "application",
		modulePrefix + "interfaces",
	},
}

// TestLayerDependencies 防止分层依赖方向被破坏（测试文件除外）。
func TestLayerDependencies(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	root := filepath.Dir(filename)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		var forbidden []string
		for layer, rules := range layerRules {
			if strings.HasPrefix(rel, layer) {
				forbidden = append(forbidden, rules...)
			}
		}
		if len(forbidden) == 0 {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imp, _ := strconv.Unquote(spec.Path.Value)
			for _, prefix := range forbidden {
				if strings.HasPrefix(imp, prefix) {
					t.Errorf("%s imports %q, which violates layer rules", rel, imp)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

const (
	goRedisImport  = "github.com/redis/go-redis/v9"
	redisKeyImport = modulePrefix + "pkg/rediskey"
)

// redisWithoutKeys 列出使用 Redis 客户端却不读写任何 key 的文件（连通性探测、关闭连接）。
var redisWithoutKeys = map[string]string{
	"interfaces/api/router.go":        "readiness probe PINGs Redis",
	"interfaces/api/server/server.go": "closes the client on shutdown",
}

// TestRedisKeysNamespaced 防止新代码绕开实例命名空间直接读写 Redis：多套部署共用一个 Redis 时，
// 没有前缀的 key 会被所有实例共享（RSA 密钥互相覆盖、字典串台、限流互相锁定）。
// 使用 go-redis 的文件必须同时引入 rediskey，并用 Namespace.Key 构造每一个 key。
func TestRedisKeysNamespaced(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	root := filepath.Dir(filename)
	seen := map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		var usesRedis, usesNamespace bool
		for _, spec := range file.Imports {
			imp, _ := strconv.Unquote(spec.Path.Value)
			usesRedis = usesRedis || imp == goRedisImport
			usesNamespace = usesNamespace || imp == redisKeyImport
		}
		if !usesRedis {
			return nil
		}
		if _, exempt := redisWithoutKeys[rel]; exempt {
			seen[rel] = true
			return nil
		}
		if !usesNamespace {
			t.Errorf("%s uses Redis without %s: build every key with rediskey.Namespace.Key, "+
				"or list the file in redisWithoutKeys if it never touches a key", rel, redisKeyImport)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for rel := range redisWithoutKeys {
		if !seen[rel] {
			t.Errorf("redisWithoutKeys lists %s, which no longer uses Redis; remove the entry", rel)
		}
	}
}
