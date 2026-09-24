package bootstrap

import (
	"strings"

	"github.com/castorworks/castor/internal/domain/menu"
	"github.com/castorworks/castor/internal/domain/permission"
)

// desiredMenu 是系统内置导航树中的一个期望节点。
// ParentCode 为空表示根节点；切片中父节点永远排在子节点之前，
// 这样一次顺序遍历就能把缺失节点连同其父节点一并补齐。
type desiredMenu struct {
	menu.Menu
	ParentCode string
}

// systemMenuTree 根据资源目录推导出完整的系统菜单树（目录 + 页面 + 按钮级 action 节点）。
// 这是新库播种与老库增量补齐共用的唯一声明，新增模块只需在这里追加节点。
func systemMenuTree(resources []permission.Resource) []desiredMenu {
	byPath := make(map[string]permission.Resource, len(resources))
	for _, r := range resources {
		if r.Actions.Contains("GET") {
			byPath[r.Path] = r
		}
	}
	permissionOf := func(path string) []menu.Permission {
		return []menu.Permission{{ResourceID: byPath[path].ID, Action: "GET"}}
	}
	node := func(parent string, m menu.Menu) desiredMenu {
		m.IsEnabled = true
		return desiredMenu{Menu: m, ParentCode: parent}
	}

	desired := []desiredMenu{
		// 一级目录。
		node("", menu.Menu{Code: "overview", Kind: menu.Directory, Titles: menu.Titles{En: "Overview", Zh: "概览", Ja: "概要", Ko: "개요"}, SortOrder: 0, AccessMode: "authenticated"}),
		node("", menu.Menu{Code: "system", Kind: menu.Directory, Titles: menu.Titles{En: "System Management", Zh: "系统管理", Ja: "システム管理", Ko: "시스템 관리"}, SortOrder: 10, AccessMode: "authenticated"}),
		node("", menu.Menu{Code: "account", Kind: menu.Directory, Titles: menu.Titles{En: "Account", Zh: "账户", Ja: "アカウント", Ko: "계정"}, SortOrder: 20, AccessMode: "authenticated"}),
		// 系统管理下的分组目录。
		node("system", menu.Menu{Code: "system_access", Kind: menu.Directory, Titles: menu.Titles{En: "Access Control", Zh: "访问控制", Ja: "アクセス制御", Ko: "접근 제어"}, Icon: "shield", SortOrder: 10, AccessMode: "authenticated"}),
		node("system", menu.Menu{Code: "system_configuration", Kind: menu.Directory, Titles: menu.Titles{En: "System Configuration", Zh: "系统配置", Ja: "システム設定", Ko: "시스템 설정"}, Icon: "settings", SortOrder: 20, AccessMode: "authenticated"}),
		node("system", menu.Menu{Code: "system_operations", Kind: menu.Directory, Titles: menu.Titles{En: "Operations", Zh: "运营管理", Ja: "運用管理", Ko: "운영 관리"}, Icon: "workspace", SortOrder: 30, AccessMode: "authenticated"}),
		node("system", menu.Menu{Code: "system_audit", Kind: menu.Directory, Titles: menu.Titles{En: "Audit & Security", Zh: "审计与安全", Ja: "監査とセキュリティ", Ko: "감사 및 보안"}, Icon: "clock", SortOrder: 40, AccessMode: "authenticated"}),
		// 页面节点。
		node("overview", menu.Menu{Code: "page_dashboard", Kind: menu.Page, Titles: menu.Titles{En: "Dashboard", Zh: "仪表盘", Ja: "ダッシュボード", Ko: "대시보드"}, Path: "/dashboard/overview", Icon: "dashboard", SortOrder: 0, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/dashboard/stats")}),
		node("overview", menu.Menu{Code: "page_notifications", Kind: menu.Page, Titles: menu.Titles{En: "Notifications", Zh: "通知", Ja: "通知", Ko: "알림"}, Path: "/dashboard/notifications", Icon: "notification", SortOrder: 10, AccessMode: "authenticated", Permissions: []menu.Permission{}}),
		node("system_access", menu.Menu{Code: "page_users", Kind: menu.Page, Titles: menu.Titles{En: "User Management", Zh: "用户管理", Ja: "ユーザー管理", Ko: "사용자 관리"}, Path: "/dashboard/users", Icon: "teams", SortOrder: 10, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/users")}),
		node("system_access", menu.Menu{Code: "page_roles", Kind: menu.Page, Titles: menu.Titles{En: "Role Management", Zh: "角色管理", Ja: "ロール管理", Ko: "역할 관리"}, Path: "/dashboard/roles", Icon: "lock", SortOrder: 20, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/roles")}),
		node("system_access", menu.Menu{Code: "page_resources", Kind: menu.Page, Titles: menu.Titles{En: "Resource Management", Zh: "资源管理", Ja: "リソース管理", Ko: "리소스 관리"}, Path: "/dashboard/resources", Icon: "shield", SortOrder: 40, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/resources")}),
		node("system_access", menu.Menu{Code: "page_authorization", Kind: menu.Page, Titles: menu.Titles{En: "Authorization Constraints", Zh: "授权约束", Ja: "認可制約", Ko: "권한 제약"}, Path: "/dashboard/authorization", Icon: "shield", SortOrder: 50, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/authorization/constraints")}),
		node("system_access", menu.Menu{Code: "page_departments", Kind: menu.Page, Titles: menu.Titles{En: "Departments", Zh: "部门管理", Ja: "部署管理", Ko: "부서 관리"}, Path: "/dashboard/departments", Icon: "employee", SortOrder: 15, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/departments")}),
		node("system_access", menu.Menu{Code: "page_menus", Kind: menu.Page, Titles: menu.Titles{En: "Menus", Zh: "菜单管理", Ja: "メニュー管理", Ko: "메뉴 관리"}, Path: "/dashboard/menus", Icon: "panelLeft", SortOrder: 30, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/menus")}),
		node("system_operations", menu.Menu{Code: "page_assets", Kind: menu.Page, Titles: menu.Titles{En: "Asset Management", Zh: "资产管理", Ja: "アセット管理", Ko: "자산 관리"}, Path: "/dashboard/assets", Icon: "workspace", SortOrder: 10, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/assets")}),
		node("system_configuration", menu.Menu{Code: "page_dictionaries", Kind: menu.Page, Titles: menu.Titles{En: "Data Dictionary", Zh: "数据字典", Ja: "データディクショナリ", Ko: "데이터 사전"}, Path: "/dashboard/dictionaries", Icon: "code", SortOrder: 20, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/dict-types")}),
		node("system_configuration", menu.Menu{Code: "page_sso", Kind: menu.Page, Titles: menu.Titles{En: "Single Sign-On", Zh: "单点登录", Ja: "シングルサインオン", Ko: "싱글 사인온"}, Path: "/dashboard/sso", Icon: "shield", SortOrder: 25, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/oidc-providers")}),
		node("system_configuration", menu.Menu{Code: "page_api_docs", Kind: menu.Page, Titles: menu.Titles{En: "API Docs", Zh: "接口文档", Ja: "API ドキュメント", Ko: "API 문서"}, Path: "/dashboard/api-docs", Icon: "code", SortOrder: 30, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/openapi.json")}),
		node("system_configuration", menu.Menu{Code: "page_settings", Kind: menu.Page, Titles: menu.Titles{En: "System Settings", Zh: "系统设置", Ja: "システム設定", Ko: "시스템 설정"}, Path: "/dashboard/settings", Icon: "settings", SortOrder: 10, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/settings")}),
		node("system_operations", menu.Menu{Code: "page_admin_notifications", Kind: menu.Page, Titles: menu.Titles{En: "Notification Management", Zh: "通知管理", Ja: "通知管理", Ko: "알림 관리"}, Path: "/dashboard/admin-notifications", Icon: "notification", SortOrder: 20, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/notifications")}),
		node("system_operations", menu.Menu{Code: "page_jobs", Kind: menu.Page, Titles: menu.Titles{En: "Scheduled Jobs", Zh: "定时任务", Ja: "スケジュールジョブ", Ko: "예약 작업"}, Path: "/dashboard/jobs", Icon: "calendar", SortOrder: 30, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/jobs")}),
		node("system_audit", menu.Menu{Code: "page_audit_logs", Kind: menu.Page, Titles: menu.Titles{En: "Audit Logs", Zh: "审计日志", Ja: "監査ログ", Ko: "감사 로그"}, Path: "/dashboard/audit-logs", Icon: "clock", SortOrder: 10, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/audit-logs")}),
		node("system_audit", menu.Menu{Code: "page_sessions", Kind: menu.Page, Titles: menu.Titles{En: "Online Sessions", Zh: "在线会话", Ja: "オンラインセッション", Ko: "온라인 세션"}, Path: "/dashboard/sessions", Icon: "laptop", SortOrder: 30, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/sessions")}),
		node("system_audit", menu.Menu{Code: "page_login_histories", Kind: menu.Page, Titles: menu.Titles{En: "Login Histories", Zh: "登录历史", Ja: "ログイン履歴", Ko: "로그인 기록"}, Path: "/dashboard/login-histories", Icon: "login", SortOrder: 20, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/login-histories")}),
		node("account", menu.Menu{Code: "page_profile", Kind: menu.Page, Titles: menu.Titles{En: "Profile", Zh: "个人资料", Ja: "プロフィール", Ko: "프로필"}, Path: "/dashboard/profile", Icon: "profile", SortOrder: 130, AccessMode: "authenticated", Permissions: []menu.Permission{}}),
		node("account", menu.Menu{Code: "page_account_history", Kind: menu.Page, Titles: menu.Titles{En: "Login Histories", Zh: "登录历史", Ja: "ログイン履歴", Ko: "로그인 기록"}, Path: "/dashboard/account/login-history", Icon: "login", SortOrder: 140, AccessMode: "authenticated", Permissions: []menu.Permission{}}),
	}
	return append(desired, desiredActionMenus(resources)...)
}

// menuModules 把资源模块映射到承载其按钮级 action 节点的页面。
var menuModules = map[string]string{
	"users": "users", "roles": "roles", "resources": "resources", "authorization": "authorization",
	"menus": "menus", "assets": "assets", "dict-items": "dictionaries", "dict-types": "dictionaries",
	"dictionaries": "dictionaries", "dictionary": "dictionaries", "settings": "settings",
	"notifications": "admin_notifications", "audit-logs": "audit_logs",
	"login-histories": "login_histories", "dashboard": "dashboard", "sessions": "sessions",
	"departments": "departments", "jobs": "jobs", "openapi": "api_docs", "oidc-providers": "sso",
}

// menuPageResources 记录每个页面节点自身已经表达的列表/读取权限，
// 对应的 GET action 不再重复生成 action 节点。
var menuPageResources = map[string]string{
	"dashboard":           "/api/v1/admin/dashboard/stats",
	"users":               "/api/v1/admin/users",
	"roles":               "/api/v1/admin/roles",
	"resources":           "/api/v1/admin/resources",
	"authorization":       "/api/v1/admin/authorization/constraints",
	"menus":               "/api/v1/admin/menus",
	"assets":              "/api/v1/admin/assets",
	"dictionaries":        "/api/v1/admin/dict-types",
	"settings":            "/api/v1/admin/settings",
	"admin_notifications": "/api/v1/admin/notifications",
	"audit_logs":          "/api/v1/admin/audit-logs",
	"login_histories":     "/api/v1/admin/login-histories",
	"sessions":            "/api/v1/admin/sessions",
	"departments":         "/api/v1/admin/departments",
	"jobs":                "/api/v1/admin/jobs",
	"api_docs":            "/api/v1/admin/openapi.json",
	"sso":                 "/api/v1/admin/oidc-providers",
}

// desiredActionMenus 为每个管理类资源动作推导按钮级 action 节点。
func desiredActionMenus(resources []permission.Resource) []desiredMenu {
	var desired []desiredMenu
	for _, r := range resources {
		page, ok := menuModules[r.Module]
		if !ok || r.Category != permission.CategoryAdmin {
			continue
		}
		for _, action := range r.Actions {
			// 页面节点已经代表了列表/读取权限。
			if action == "GET" && menuPageResources[page] == r.Path {
				continue
			}
			titles, ok := defaultActionTitles[r.Name]
			if !ok {
				continue
			}
			titles.En += " · " + action
			titles.Zh += " · " + action
			titles.Ja += " · " + action
			titles.Ko += " · " + action
			code := "action_" + strings.NewReplacer(":", "_", "-", "_").Replace(r.Code) + "_" + strings.ToLower(action)
			desired = append(desired, desiredMenu{
				ParentCode: "page_" + page,
				Menu: menu.Menu{
					Code: code, Kind: menu.Action, Titles: titles, IsEnabled: true,
					AccessMode: "permission", SortOrder: r.SortOrder,
					Permissions: []menu.Permission{{ResourceID: r.ID, Action: action}},
				},
			})
		}
	}
	return desired
}

var defaultActionTitles = map[string]menu.Titles{
	"seedResources.admin.assets.batchDelete":               {En: "Batch Delete Assets", Zh: "批量删除资产", Ja: "アセット一括削除", Ko: "자산 일괄 삭제"},
	"seedResources.admin.assets.batchUpdateStatus":         {En: "Batch Update Asset Status", Zh: "批量更新资产状态", Ja: "アセットステータス一括更新", Ko: "자산 상태 일괄 수정"},
	"seedResources.admin.assets.create":                    {En: "Create Asset", Zh: "创建资产", Ja: "アセット作成", Ko: "자산 생성"},
	"seedResources.admin.assets.delete":                    {En: "Delete Asset", Zh: "删除资产", Ja: "アセット削除", Ko: "자산 삭제"},
	"seedResources.admin.assets.detail":                    {En: "Asset Detail", Zh: "资产详情", Ja: "アセット詳細", Ko: "자산 상세"},
	"seedResources.admin.assets.download":                  {En: "Download Asset", Zh: "下载资产", Ja: "アセットをダウンロード", Ko: "자산 다운로드"},
	"seedResources.admin.assets.list":                      {En: "List Assets", Zh: "资产列表", Ja: "アセット一覧", Ko: "자산 목록"},
	"seedResources.admin.assets.move":                      {En: "Move Asset", Zh: "移动资产", Ja: "アセット移動", Ko: "자산 이동"},
	"seedResources.admin.assets.stats":                     {En: "Asset Statistics", Zh: "资产统计", Ja: "アセット統計", Ko: "자산 통계"},
	"seedResources.admin.assets.update":                    {En: "Update Asset", Zh: "更新资产", Ja: "アセット更新", Ko: "자산 수정"},
	"seedResources.admin.assets.updateStatus":              {En: "Update Asset Status", Zh: "更新资产状态", Ja: "アセットステータス更新", Ko: "자산 상태 수정"},
	"seedResources.admin.auditLogs.cleanup":                {En: "Cleanup Audit Logs", Zh: "清理审计日志", Ja: "監査ログクリーンアップ", Ko: "감사 로그 정리"},
	"seedResources.admin.auditLogs.list":                   {En: "List Audit Logs", Zh: "审计日志列表", Ja: "監査ログ一覧", Ko: "감사 로그 목록"},
	"seedResources.admin.authorization.actions":            {En: "Get Actions", Zh: "获取操作", Ja: "アクション取得", Ko: "액션 조회"},
	"seedResources.admin.authorization.addUserRole":        {En: "Add User Role", Zh: "添加用户角色", Ja: "ユーザーロール追加", Ko: "사용자 역할 추가"},
	"seedResources.admin.authorization.constraint":         {En: "Separation of Duty Constraint", Zh: "职责分离约束", Ja: "職務分離制約", Ko: "직무 분리 제약"},
	"seedResources.admin.authorization.constraints":        {En: "Separation of Duty Constraints", Zh: "职责分离约束", Ja: "職務分離制約", Ko: "직무 분리 제약"},
	"seedResources.admin.authorization.deleteUserRole":     {En: "Delete User Role", Zh: "删除用户角色", Ja: "ユーザーロール削除", Ko: "사용자 역할 삭제"},
	"seedResources.admin.authorization.objects":            {En: "Get Objects", Zh: "获取对象", Ja: "オブジェクト取得", Ko: "객체 조회"},
	"seedResources.admin.authorization.policies":           {En: "Get Policies", Zh: "获取策略", Ja: "ポリシー取得", Ko: "정책 조회"},
	"seedResources.admin.authorization.subjects":           {En: "Get Subjects", Zh: "获取主体", Ja: "サブジェクト取得", Ko: "주체 조회"},
	"seedResources.admin.authorization.userPermissions":    {En: "Get User Permissions", Zh: "获取用户权限", Ja: "ユーザー権限取得", Ko: "사용자 권한 조회"},
	"seedResources.admin.authorization.userRoles":          {En: "Get User Roles", Zh: "获取用户角色", Ja: "ユーザーロール取得", Ko: "사용자 역할 조회"},
	"seedResources.admin.dashboard.stats":                  {En: "Dashboard Statistics", Zh: "仪表盘统计", Ja: "ダッシュボード統計", Ko: "대시보드 통계"},
	"seedResources.admin.dictItems.create":                 {En: "Create Dict Item", Zh: "创建字典项", Ja: "辞書項目作成", Ko: "사전 항목 생성"},
	"seedResources.admin.dictItems.delete":                 {En: "Delete Dict Item", Zh: "删除字典项", Ja: "辞書項目削除", Ko: "사전 항목 삭제"},
	"seedResources.admin.dictItems.detail":                 {En: "Dict Item Detail", Zh: "字典项详情", Ja: "辞書項目詳細", Ko: "사전 항목 상세"},
	"seedResources.admin.dictItems.list":                   {En: "List Dict Items", Zh: "字典项列表", Ja: "辞書項目一覧", Ko: "사전 항목 목록"},
	"seedResources.admin.dictItems.listByType":             {En: "List Dict Items by Type", Zh: "按类型查看字典项", Ja: "タイプ別辞書項目一覧", Ko: "유형별 사전 항목 목록"},
	"seedResources.admin.dictItems.update":                 {En: "Update Dict Item", Zh: "更新字典项", Ja: "辞書項目更新", Ko: "사전 항목 수정"},
	"seedResources.admin.dictTypes.create":                 {En: "Create Dict Type", Zh: "创建字典类型", Ja: "辞書タイプ作成", Ko: "사전 유형 생성"},
	"seedResources.admin.dictTypes.delete":                 {En: "Delete Dict Type", Zh: "删除字典类型", Ja: "辞書タイプ削除", Ko: "사전 유형 삭제"},
	"seedResources.admin.dictTypes.detail":                 {En: "Dict Type Detail", Zh: "字典类型详情", Ja: "辞書タイプ詳細", Ko: "사전 유형 상세"},
	"seedResources.admin.dictTypes.list":                   {En: "List Dict Types", Zh: "字典类型列表", Ja: "辞書タイプ一覧", Ko: "사전 유형 목록"},
	"seedResources.admin.dictTypes.update":                 {En: "Update Dict Type", Zh: "更新字典类型", Ja: "辞書タイプ更新", Ko: "사전 유형 수정"},
	"seedResources.admin.loginHistories.batchDelete":       {En: "Batch Delete Login Histories", Zh: "批量删除登录历史", Ja: "ログイン履歴一括削除", Ko: "로그인 기록 일괄 삭제"},
	"seedResources.admin.loginHistories.delete":            {En: "Delete Login History", Zh: "删除登录历史", Ja: "ログイン履歴削除", Ko: "로그인 기록 삭제"},
	"seedResources.admin.loginHistories.list":              {En: "List Login Histories", Zh: "登录历史列表", Ja: "ログイン履歴一覧", Ko: "로그인 기록 목록"},
	"seedResources.admin.departments.create":               {En: "Create Department", Zh: "创建部门", Ja: "部署作成", Ko: "부서 생성"},
	"seedResources.admin.departments.delete":               {En: "Delete Department", Zh: "删除部门", Ja: "部署削除", Ko: "부서 삭제"},
	"seedResources.admin.departments.list":                 {En: "List Departments", Zh: "查看部门", Ja: "部署一覧", Ko: "부서 목록"},
	"seedResources.admin.departments.update":               {En: "Update Department", Zh: "修改部门", Ja: "部署更新", Ko: "부서 수정"},
	"seedResources.admin.users.export":                     {En: "Export Users", Zh: "导出用户", Ja: "ユーザーエクスポート", Ko: "사용자 내보내기"},
	"seedResources.admin.users.importTemplate":             {En: "Download User Import Template", Zh: "下载用户导入模板", Ja: "ユーザーインポート用テンプレートのダウンロード", Ko: "사용자 가져오기 템플릿 다운로드"},
	"seedResources.admin.users.import":                     {En: "Import Users", Zh: "导入用户", Ja: "ユーザーインポート", Ko: "사용자 가져오기"},
	"seedResources.admin.loginHistories.export":            {En: "Export Login Histories", Zh: "导出登录历史", Ja: "ログイン履歴エクスポート", Ko: "로그인 기록 내보내기"},
	"seedResources.admin.auditLogs.export":                 {En: "Export Audit Logs", Zh: "导出审计日志", Ja: "監査ログエクスポート", Ko: "감사 로그 내보내기"},
	"seedResources.admin.users.resetTotp":                  {En: "Reset User Two-factor", Zh: "重置用户两步验证", Ja: "ユーザーの 2 段階認証リセット", Ko: "사용자 2단계 인증 초기화"},
	"seedResources.admin.oidcProviders.list":               {En: "List Identity Providers", Zh: "查看身份提供方", Ja: "ID プロバイダー一覧", Ko: "ID 공급자 목록"},
	"seedResources.admin.oidcProviders.create":             {En: "Create Identity Provider", Zh: "新建身份提供方", Ja: "ID プロバイダー作成", Ko: "ID 공급자 생성"},
	"seedResources.admin.oidcProviders.update":             {En: "Update Identity Provider", Zh: "修改身份提供方", Ja: "ID プロバイダー更新", Ko: "ID 공급자 수정"},
	"seedResources.admin.oidcProviders.delete":             {En: "Delete Identity Provider", Zh: "删除身份提供方", Ja: "ID プロバイダー削除", Ko: "ID 공급자 삭제"},
	"seedResources.admin.notifications.channels":           {En: "View Notification Channels", Zh: "查看通知投递通道", Ja: "通知の配信チャネル参照", Ko: "알림 전송 채널 보기"},
	"seedResources.admin.openapi.get":                      {En: "View API Docs", Zh: "查看接口文档", Ja: "API ドキュメントの閲覧", Ko: "API 문서 보기"},
	"seedResources.admin.jobs.list":                        {En: "List Scheduled Jobs", Zh: "查看定时任务", Ja: "スケジュールジョブ一覧", Ko: "예약 작업 목록"},
	"seedResources.admin.jobs.update":                      {En: "Update Scheduled Job", Zh: "修改定时任务", Ja: "スケジュールジョブ更新", Ko: "예약 작업 수정"},
	"seedResources.admin.jobs.run":                         {En: "Run Scheduled Job", Zh: "手动执行定时任务", Ja: "スケジュールジョブ実行", Ko: "예약 작업 실행"},
	"seedResources.admin.jobRuns.list":                     {En: "List Job Runs", Zh: "查看任务执行记录", Ja: "ジョブ実行履歴一覧", Ko: "작업 실행 기록 목록"},
	"seedResources.admin.roles.dataScope":                  {En: "Set Role Data Scope", Zh: "设置角色数据范围", Ja: "ロールのデータ範囲設定", Ko: "역할 데이터 범위 설정"},
	"seedResources.admin.sessions.list":                    {En: "List Online Sessions", Zh: "查看在线会话", Ja: "オンラインセッション一覧", Ko: "온라인 세션 목록"},
	"seedResources.admin.sessions.revoke":                  {En: "Revoke Session", Zh: "强制下线", Ja: "強制サインアウト", Ko: "강제 로그아웃"},
	"seedResources.admin.menus.create":                     {En: "Create Menu", Zh: "创建菜单", Ja: "メニュー作成", Ko: "메뉴 생성"},
	"seedResources.admin.menus.delete":                     {En: "Delete Menu", Zh: "删除菜单", Ja: "メニュー削除", Ko: "메뉴 삭제"},
	"seedResources.admin.menus.list":                       {En: "List Menus", Zh: "查看菜单", Ja: "メニュー一覧", Ko: "메뉴 목록"},
	"seedResources.admin.menus.update":                     {En: "Update Menu", Zh: "修改菜单", Ja: "メニュー更新", Ko: "메뉴 수정"},
	"seedResources.admin.notifications.batchDelete":        {En: "Batch Delete Notifications", Zh: "批量删除通知", Ja: "通知一括削除", Ko: "알림 일괄 삭제"},
	"seedResources.admin.notifications.create":             {En: "Create Notification", Zh: "创建通知", Ja: "通知作成", Ko: "알림 생성"},
	"seedResources.admin.notifications.delete":             {En: "Delete Notification", Zh: "删除通知", Ja: "通知削除", Ko: "알림 삭제"},
	"seedResources.admin.notifications.detail":             {En: "Notification Detail", Zh: "通知详情", Ja: "通知詳細", Ko: "알림 상세"},
	"seedResources.admin.notifications.list":               {En: "List Notifications", Zh: "通知列表", Ja: "通知一覧", Ko: "알림 목록"},
	"seedResources.admin.notifications.recipients":         {En: "Notification Recipients", Zh: "通知接收人", Ja: "通知受信者", Ko: "알림 수신자"},
	"seedResources.admin.notifications.uploadAttachment":   {En: "Upload Notification Attachment", Zh: "上传通知附件", Ja: "通知の添付ファイルをアップロード", Ko: "알림 첨부 파일 업로드"},
	"seedResources.admin.notifications.downloadAttachment": {En: "Download Notification Attachment", Zh: "下载通知附件", Ja: "通知の添付ファイルをダウンロード", Ko: "알림 첨부 파일 다운로드"},
	"seedResources.admin.notifications.update":             {En: "Update Notification", Zh: "更新通知", Ja: "通知更新", Ko: "알림 수정"},
	"seedResources.admin.resources.create":                 {En: "Create Resource", Zh: "创建资源", Ja: "リソース作成", Ko: "리소스 생성"},
	"seedResources.admin.resources.delete":                 {En: "Delete Resource", Zh: "删除资源", Ja: "リソース削除", Ko: "리소스 삭제"},
	"seedResources.admin.resources.detail":                 {En: "Resource Detail", Zh: "资源详情", Ja: "リソース詳細", Ko: "리소스 상세"},
	"seedResources.admin.resources.list":                   {En: "List Resources", Zh: "资源列表", Ja: "リソース一覧", Ko: "리소스 목록"},
	"seedResources.admin.resources.modules":                {En: "Resource Modules", Zh: "资源模块", Ja: "リソースモジュール", Ko: "리소스 모듈"},
	"seedResources.admin.resources.update":                 {En: "Update Resource", Zh: "更新资源", Ja: "リソース更新", Ko: "리소스 수정"},
	"seedResources.admin.roles.create":                     {En: "Create Role", Zh: "创建角色", Ja: "ロール作成", Ko: "역할 생성"},
	"seedResources.admin.roles.delete":                     {En: "Delete Role", Zh: "删除角色", Ja: "ロール削除", Ko: "역할 삭제"},
	"seedResources.admin.roles.detail":                     {En: "Role Detail", Zh: "角色详情", Ja: "ロール詳細", Ko: "역할 상세"},
	"seedResources.admin.roles.hierarchy":                  {En: "Role Hierarchy", Zh: "角色继承", Ja: "ロール階層", Ko: "역할 계층"},
	"seedResources.admin.roles.list":                       {En: "List Roles", Zh: "角色列表", Ja: "ロール一覧", Ko: "역할 목록"},
	"seedResources.admin.roles.permissions":                {En: "Role Permissions", Zh: "角色权限", Ja: "ロール権限", Ko: "역할 권한"},
	"seedResources.admin.roles.setPermissions":             {En: "Set Role Permissions", Zh: "设置角色权限", Ja: "ロール権限設定", Ko: "역할 권한 설정"},
	"seedResources.admin.roles.update":                     {En: "Update Role", Zh: "更新角色", Ja: "ロール更新", Ko: "역할 수정"},
	"seedResources.admin.roles.users":                      {En: "Role Users", Zh: "角色用户", Ja: "ロールユーザー", Ko: "역할 사용자"},
	"seedResources.admin.settings.batchUpdate":             {En: "Batch Update Settings", Zh: "批量更新设置", Ja: "設定一括更新", Ko: "설정 일괄 수정"},
	"seedResources.admin.settings.create":                  {En: "Create Setting", Zh: "创建设置", Ja: "設定作成", Ko: "설정 생성"},
	"seedResources.admin.settings.delete":                  {En: "Delete Setting", Zh: "删除设置", Ja: "設定削除", Ko: "설정 삭제"},
	"seedResources.admin.settings.detail":                  {En: "Setting Detail", Zh: "设置详情", Ja: "設定詳細", Ko: "설정 상세"},
	"seedResources.admin.settings.getByKey":                {En: "Get Setting by Key", Zh: "按键查询设置", Ja: "キーで設定取得", Ko: "키로 설정 조회"},
	"seedResources.admin.settings.list":                    {En: "List Settings", Zh: "系统设置列表", Ja: "設定一覧", Ko: "설정 목록"},
	"seedResources.admin.settings.update":                  {En: "Update Setting", Zh: "更新设置", Ja: "設定更新", Ko: "설정 수정"},
	"seedResources.admin.users.create":                     {En: "Create User", Zh: "创建用户", Ja: "ユーザー作成", Ko: "사용자 생성"},
	"seedResources.admin.users.delete":                     {En: "Delete User", Zh: "删除用户", Ja: "ユーザー削除", Ko: "사용자 삭제"},
	"seedResources.admin.users.detail":                     {En: "User Detail", Zh: "用户详情", Ja: "ユーザー詳細", Ko: "사용자 상세"},
	"seedResources.admin.users.list":                       {En: "List Users", Zh: "用户列表", Ja: "ユーザー一覧", Ko: "사용자 목록"},
	"seedResources.admin.users.update":                     {En: "Update User", Zh: "更新用户", Ja: "ユーザー更新", Ko: "사용자 수정"},
}
