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
