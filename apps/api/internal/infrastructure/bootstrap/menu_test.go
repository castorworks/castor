package bootstrap

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/domain/menu"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/infrastructure/persistence"
)

// systemMenuTree 的声明顺序是补齐算法的前提：父节点必须先于子节点出现，
// 否则老库上补建的新页面会挂不上父目录。
func TestSystemMenuTreeDeclaration(t *testing.T) {
	desired := systemMenuTree(permission.DefaultResources)
	if len(desired) == 0 {
		t.Fatal("desired tree is empty")
	}
	seen := make(map[string]desiredMenu, len(desired))
	for _, want := range desired {
		if _, dup := seen[want.Code]; dup {
			t.Fatalf("duplicate menu code %q", want.Code)
		}
		if want.Code == "" || want.Kind == "" || want.Titles.En == "" {
			t.Fatalf("incomplete menu declaration: %+v", want.Menu)
		}
		if !want.IsEnabled {
			t.Fatalf("menu %q is declared disabled", want.Code)
		}
		if want.ID != 0 || want.ParentID != nil {
			t.Fatalf("menu %q must not declare database identifiers", want.Code)
		}
		if want.ParentCode != "" {
			if _, ok := seen[want.ParentCode]; !ok {
				t.Fatalf("menu %q declares parent %q that is not declared before it", want.Code, want.ParentCode)
			}
		}
		seen[want.Code] = want
	}
	for _, page := range menuModules {
		if _, ok := seen["page_"+page]; !ok {
			t.Fatalf("module maps to page_%s which the tree does not declare", page)
		}
	}
}

// 需要一次性数据库（CASTOR_TEST_POSTGRES_DSN），绝不能指向开发或生产库。
func TestInitializeMenusIsReentrant(t *testing.T) {
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	t.Setenv("CASTOR_DEFAULT_ADMIN_PASSWORD", "InitialPassword123!")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	// 先完成一次完整初始化，拿到资源目录与表结构。
	_, err = Initialize(ctx, db)
	must(err)

	menus := persistence.NewMenuRepository(db)
	resources := persistence.NewResourceRepository(db)
	reseed := func() { t.Helper(); must(initializeMenus(ctx, menus, resources)) }
	load := func() (map[string]menu.Menu, map[uint]string) {
		t.Helper()
		items, err := menus.List(ctx)
		must(err)
		byCode := make(map[string]menu.Menu, len(items))
		byID := make(map[uint]string, len(items))
		for _, item := range items {
			byCode[item.Code] = item
			byID[item.ID] = item.Code
		}
		return byCode, byID
	}
	parentOf := func(byCode map[string]menu.Menu, byID map[uint]string, code string) string {
		t.Helper()
		item, ok := byCode[code]
		if !ok {
			t.Fatalf("menu %q is missing", code)
		}
		if item.ParentID == nil {
			return ""
		}
		return byID[*item.ParentID]
	}

	// 全新安装：清空菜单后重新播种，树形必须与期望声明完全一致。
	must(db.WithContext(ctx).Exec("TRUNCATE rbac_menu_permissions, rbac_menus RESTART IDENTITY CASCADE").Error)
	reseed()
	byCode, byID := load()
	catalog, err := resources.GetAll(ctx)
	must(err)
	for _, want := range systemMenuTree(catalog) {
		got, ok := byCode[want.Code]
		if !ok {
			t.Fatalf("fresh install is missing menu %q", want.Code)
		}
		if got.Kind != want.Kind || got.Path != want.Path || got.Icon != want.Icon ||
			got.SortOrder != want.SortOrder || got.AccessMode != want.AccessMode || got.Titles != want.Titles {
			t.Fatalf("fresh install menu %q mismatch: %+v", want.Code, got)
		}
		if parentOf(byCode, byID, want.Code) != want.ParentCode {
			t.Fatalf("fresh install menu %q parent=%q want %q", want.Code, parentOf(byCode, byID, want.Code), want.ParentCode)
		}
	}
	for _, pair := range [][2]string{
		{"system_access", "system"}, {"system_configuration", "system"},
		{"system_operations", "system"}, {"system_audit", "system"},
		{"page_users", "system_access"}, {"page_settings", "system_configuration"},
		{"page_assets", "system_operations"}, {"page_audit_logs", "system_audit"},
		{"page_dashboard", "overview"}, {"page_profile", "account"},
	} {
		if got := parentOf(byCode, byID, pair[0]); got != pair[1] {
			t.Fatalf("%s parent=%q want %q", pair[0], got, pair[1])
		}
	}
	freshCount := len(byCode)

	// 幂等：紧接着再跑一次不应产生任何节点。
	reseed()
	byCode, byID = load()
	if len(byCode) != freshCount {
		t.Fatalf("second run changed menu count: %d -> %d", freshCount, len(byCode))
	}

	// 回归：老库上缺失的页面及其按钮节点必须被重新补齐。
	// 这正是新模块上线后已安装环境拿不到导航入口的缺陷。
	var removed []string
	for code := range byCode {
		if strings.HasPrefix(code, "action_admin_menus_") {
			removed = append(removed, code)
		}
	}
	if len(removed) == 0 {
		t.Fatal("expected action nodes under page_menus")
	}
	// 先删子节点再删页面，外键不允许留下孤儿。
	sort.Strings(removed)
	removed = append(removed, "page_menus")
	for _, code := range removed {
		must(menus.Delete(ctx, byCode[code].ID))
	}
	reseed()
	byCode, byID = load()
	for _, code := range removed {
		if _, ok := byCode[code]; !ok {
			t.Fatalf("re-run did not recreate menu %q", code)
		}
	}
	if got := parentOf(byCode, byID, "page_menus"); got != "system_access" {
		t.Fatalf("recreated page_menus parent=%q want system_access", got)
	}
	if page := byCode["page_menus"]; page.Path != "/dashboard/menus" || len(page.Permissions) != 1 || page.Permissions[0].Action != "GET" {
		t.Fatalf("recreated page_menus is incomplete: %+v", page)
	}
	if len(byCode) != freshCount {
		t.Fatalf("menu count after repair=%d want %d", len(byCode), freshCount)
	}

	// 新模块上线：声明中新增的节点连同缺失的父目录一起补建。
	must(menus.WithTx(ctx, func(tx menu.Repository) error {
		existing, err := tx.List(ctx)
		if err != nil {
			return err
		}
		return createMissingMenus(ctx, tx, existing, []desiredMenu{
			{ParentCode: "system", Menu: menu.Menu{Code: "test_group_widgets", Kind: menu.Directory, Titles: menu.Titles{En: "Widgets", Zh: "组件", Ja: "ウィジェット", Ko: "위젯"}, SortOrder: 90, IsEnabled: true, AccessMode: "authenticated"}},
			{ParentCode: "test_group_widgets", Menu: menu.Menu{Code: "test_page_widgets", Kind: menu.Page, Titles: menu.Titles{En: "Widgets", Zh: "组件", Ja: "ウィジェット", Ko: "위젯"}, Path: "/dashboard/widgets", SortOrder: 10, IsEnabled: true, AccessMode: "authenticated"}},
		})
	}))
	byCode, byID = load()
	if got := parentOf(byCode, byID, "test_page_widgets"); got != "test_group_widgets" {
		t.Fatalf("new page parent=%q want test_group_widgets", got)
	}
	if got := parentOf(byCode, byID, "test_group_widgets"); got != "system" {
		t.Fatalf("new group parent=%q want system", got)
	}
	must(menus.Delete(ctx, byCode["test_page_widgets"].ID))
	must(menus.Delete(ctx, byCode["test_group_widgets"].ID))

	// 运营人员的自定义必须在补齐后原样保留。
	byCode, _ = load()
	renamed := byCode["page_dashboard"]
	renamed.Titles.En = "Operations Console"
	renamed.Titles.Zh = "运营台"
	must(menus.Save(ctx, &renamed))
	disabled := byCode["page_notifications"]
	disabled.IsEnabled = false
	must(menus.Save(ctx, &disabled))
	// 父节点由运营人员决定，重跑 init-db 不会挪回去。
	moved := byCode["page_profile"]
	overviewID := byCode["overview"].ID
	moved.ParentID = &overviewID
	moved.SortOrder = 999
	moved.Icon = "workspace"
	must(menus.Save(ctx, &moved))

	reseed()
	byCode, byID = load()
	if got := byCode["page_dashboard"].Titles; got.En != "Operations Console" || got.Zh != "运营台" {
		t.Fatalf("re-run overwrote a renamed menu: %+v", got)
	}
	if byCode["page_notifications"].IsEnabled {
		t.Fatal("re-run re-enabled a disabled menu")
	}
	if got := parentOf(byCode, byID, "page_profile"); got != "overview" {
		t.Fatalf("re-run moved a customized menu back: parent=%q", got)
	}
	if got := byCode["page_profile"]; got.SortOrder != 999 || got.Icon != "workspace" {
		t.Fatalf("re-run overwrote menu metadata: %+v", got)
	}
	if len(byCode) != freshCount {
		t.Fatalf("menu count after customization run=%d want %d", len(byCode), freshCount)
	}

}

// TestEveryAdminModuleReachesTheMenuTree 防止新模块"接口通了但导航里什么都没有"。
//
// desiredActionMenus 在两处 map 未命中时都是静默 continue：
// 模块不在 menuModules 里，整个模块的按钮级 action 节点消失；
// 资源名不在 defaultActionTitles 里，那一个 action 节点消失。
// 两种情况都不报错、不影响启动，只是运营在「菜单管理」里永远看不到这些条目，
// 于是无法把权限授给非管理员角色。
//
// 这条测试让"漏登记"在 CI 立刻变红：新增 /admin 资源时必须同时补
// menuModules（模块 → 承载页面）与 defaultActionTitles（四语言标题）。
func TestEveryAdminModuleReachesTheMenuTree(t *testing.T) {
	var unmappedModules []string
	var untitledResources []string
	seenModule := make(map[string]bool)

	for _, resource := range permission.DefaultResources {
		if resource.Category != permission.CategoryAdmin {
			continue
		}
		if _, ok := menuModules[resource.Module]; !ok {
			if !seenModule[resource.Module] {
				seenModule[resource.Module] = true
				unmappedModules = append(unmappedModules, resource.Module)
			}
			// 模块整体没落地，逐条资源再报一遍只是噪音。
			continue
		}
		titles, ok := defaultActionTitles[resource.Name]
		if !ok {
			untitledResources = append(untitledResources, resource.Name)
			continue
		}
		if titles.En == "" || titles.Zh == "" || titles.Ja == "" || titles.Ko == "" {
			untitledResources = append(untitledResources, resource.Name+"（四语言不全）")
		}
	}

	sort.Strings(unmappedModules)
	sort.Strings(untitledResources)
	if len(unmappedModules) > 0 {
		t.Errorf("以下 admin 模块没有登记到 menuModules，其 action 菜单会被静默跳过：%v", unmappedModules)
	}
	if len(untitledResources) > 0 {
		t.Errorf("以下 admin 资源在 defaultActionTitles 中没有四语言标题，对应 action 菜单会被静默跳过：%v", untitledResources)
	}
}

// TestMenuPageResourcesMatchTheDeclaredPages 防止页面节点与其列表资源脱节。
//
// menuPageResources 告诉 desiredActionMenus："这个 GET 已经由页面节点本身表达了"。
// 漏登记不会报错，只会让页面下多出一个与页面完全重复的 GET action 节点；
// 指向一个不存在的页面或路径则说明这份映射已经过期。
func TestMenuPageResourcesMatchTheDeclaredPages(t *testing.T) {
	declared := make(map[string]bool)
	for _, want := range systemMenuTree(permission.DefaultResources) {
		declared[want.Code] = true
	}
	resourcePaths := make(map[string]bool)
	for _, resource := range permission.DefaultResources {
		resourcePaths[resource.Path] = true
	}

	for page, path := range menuPageResources {
		if !declared["page_"+page] {
			t.Errorf("menuPageResources 指向 page_%s，但菜单树没有声明该页面", page)
		}
		if !resourcePaths[path] {
			t.Errorf("menuPageResources[%q] = %q 不是任何 DefaultResources 的路径", page, path)
		}
	}

	// 每个承载 action 节点的页面都要声明自己的列表资源，
	// 否则页面节点与它的 GET action 节点会重复出现。
	var missing []string
	for _, page := range menuModules {
		if _, ok := menuPageResources[page]; !ok {
			missing = append(missing, page)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("以下页面未在 menuPageResources 声明列表资源，会生成重复的 GET action 节点：%v", missing)
	}
}
