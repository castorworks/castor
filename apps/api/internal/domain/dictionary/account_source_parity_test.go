package dictionary

import (
	"testing"

	"github.com/castorworks/castor/internal/pkg/constant"
)

// TestAccountSourcesHaveDictItems 账号来源写进 users.account_source 的值必须与字典项值逐字相同：
// 前端按字典显示标签、按字典值筛选，大小写不一致时标签退化成原值，筛选也查不到人。
func TestAccountSourcesHaveDictItems(t *testing.T) {
	seeded := map[string]bool{}
	for _, item := range DefaultDictItems {
		if item.TypeCode == "account_source" {
			seeded[item.Value] = true
		}
	}
	for _, source := range []string{constant.ACCOUNT_SOURCE_INTERNAL, constant.ACCOUNT_SOURCE_OIDC} {
		if !seeded[source] {
			t.Errorf("account source %q has no account_source dict item with the same value", source)
		}
	}
}

// TestLoginMethodsHaveDictItems 登录历史写入的登录方式也必须是字典项值，否则界面只能显示原值。
func TestLoginMethodsHaveDictItems(t *testing.T) {
	seeded := map[string]bool{}
	for _, item := range DefaultDictItems {
		if item.TypeCode == "login_method" {
			seeded[item.Value] = true
		}
	}
	for _, method := range []string{constant.LOGIN_METHOD_PASSWORD, constant.LOGIN_METHOD_EMAIL, constant.LOGIN_METHOD_MOBILE, constant.LOGIN_METHOD_OIDC} {
		if !seeded[method] {
			t.Errorf("login method %q has no login_method dict item with the same value", method)
		}
	}
}
