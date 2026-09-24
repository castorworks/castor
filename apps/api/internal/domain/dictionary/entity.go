package dictionary

import (
	"time"

	"github.com/castorworks/castor/internal/domain/shared"
)

// text 是 shared.Text 的本地简写，让下面的种子数据保持一行一条。
func text(en, zh, ja, ko string) shared.I18nText { return shared.Text(en, zh, ja, ko) }

// DictType 字典类型（纯 Domain 模型，无 ORM 依赖）
type DictType struct {
	ID          uint            `json:"id"`
	CreatedAt   time.Time       `json:"createdAt"`
	CreatedBy   uint            `json:"createdBy"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	UpdatedBy   uint            `json:"updatedBy"`
	Code        string          `json:"code"`
	Name        shared.I18nText `json:"name"`
	Description string          `json:"description"`
	IsSystem    bool            `json:"isSystem"`
	// IsPublic 决定该类型是否出现在免认证的 /api/v1/dictionaries 里。
	// 默认不公开：登录后的界面走 /api/v1/account/dictionaries 拿全部启用类型，
	// 只有登录前页面（注册、公开表单）真正需要的类型才应公开。
	IsPublic  bool `json:"isPublic"`
	IsEnabled bool `json:"isEnabled"`
	SortOrder int  `json:"sortOrder"`
}

// DictItem 字典项（纯 Domain 模型，无 ORM 依赖）
//
// Label 是四语言的：字典项由运营维护，无法预先放进 messages 文件，
// 但它会出现在徽章、下拉框、审计日志类型等大量界面上，
// 所以必须随界面语言切换，不能只存一份英文。
//
// Value 是字典项与代码里枚举常量之间唯一的连接键。系统项（IsSystem）的
// Value 不可修改、整项不可删除——否则徽章会退回原始值、筛选下拉会查不到
// 任何数据，而且全程不报错。标签、颜色、图标、排序、启停仍可由运营调整。
type DictItem struct {
	ID          uint            `json:"id"`
	CreatedAt   time.Time       `json:"createdAt"`
	CreatedBy   uint            `json:"createdBy"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	UpdatedBy   uint            `json:"updatedBy"`
	TypeCode    string          `json:"typeCode"`
	Label       shared.I18nText `json:"label"`
	Value       string          `json:"value"`
	Description string          `json:"description"`
	Color       string          `json:"color"`
	Icon        string          `json:"icon"`
	IsSystem    bool            `json:"isSystem"`
	IsDefault   bool            `json:"isDefault"`
	IsEnabled   bool            `json:"isEnabled"`
	SortOrder   int             `json:"sortOrder"`
}

// Colors 是字典项颜色的全部合法取值，与前端 `lib/tag-color.ts` 的
// `--tag-*` design token 一一对应（由前端 `lib/dict.test.ts` 把关）。
// 颜色存的是 token 名而不是色值：色值随主题与明暗模式变化，只有 token 名稳定。
var Colors = []string{"red", "orange", "yellow", "green", "blue", "purple", "pink", "gray"}

// IsValidColor 判断颜色是否合法；空串表示不着色。
func IsValidColor(color string) bool {
	if color == "" {
		return true
	}
	for _, c := range Colors {
		if c == color {
			return true
		}
	}
	return false
}

// DefaultDictTypes 默认字典类型。
//
// 代码里引用的字典类型必须在这里声明：前端 `lib/dict.ts` 的 DICT_TYPES 与本清单
// 逐项对应（`lib/dict.test.ts` 把关），运营在「字典管理」里临时建的类型不能被代码引用。
// 新增类型/字典项后重跑 init-db 即可进入已有库（见 database.SeedDictionary），无需写迁移。
var DefaultDictTypes = []DictType{
	{Code: "gender", Name: text("Gender", "性别", "性別", "성별"), IsSystem: true, IsPublic: true, IsEnabled: true, SortOrder: 1},
	{Code: "status", Name: text("Status", "状态", "ステータス", "상태"), IsSystem: true, IsPublic: true, IsEnabled: true, SortOrder: 2},
	{Code: "priority", Name: text("Priority", "优先级", "優先度", "우선순위"), IsSystem: true, IsPublic: true, IsEnabled: true, SortOrder: 3},
	{Code: "boolean", Name: text("Boolean", "布尔值", "真偽値", "불리언"), IsSystem: true, IsPublic: true, IsEnabled: true, SortOrder: 4},
	{Code: "asset_status", Name: text("Asset Status", "资产状态", "アセットステータス", "자산 상태"), IsSystem: true, IsEnabled: true, SortOrder: 5},
	{Code: "asset_category", Name: text("Asset Category", "资产分类", "アセットカテゴリ", "자산 분류"), IsSystem: true, IsEnabled: true, SortOrder: 6},
	{Code: "account_source", Name: text("Account Source", "账号来源", "アカウント種別", "계정 출처"), IsSystem: true, IsEnabled: true, SortOrder: 7},
	{Code: "notification_type", Name: text("Notification Type", "通知类型", "通知タイプ", "알림 유형"), IsSystem: true, IsEnabled: true, SortOrder: 8},
	{Code: "notification_level", Name: text("Notification Level", "通知级别", "通知レベル", "알림 수준"), IsSystem: true, IsEnabled: true, SortOrder: 9},
	{Code: "login_method", Name: text("Login Method", "登录方式", "ログイン方法", "로그인 방식"), IsSystem: true, IsEnabled: true, SortOrder: 10},
	{Code: "audit_log_type", Name: text("Audit Log Type", "审计日志类型", "監査ログタイプ", "감사 로그 유형"), IsSystem: true, IsEnabled: true, SortOrder: 11},
	{Code: "asset_scope", Name: text("Asset Scope", "资产范围", "アセット範囲", "자산 범위"), IsSystem: true, IsEnabled: true, SortOrder: 12},
	{Code: "role_data_scope", Name: text("Role Data Scope", "角色数据范围", "ロールのデータ範囲", "역할 데이터 범위"), IsSystem: true, IsEnabled: true, SortOrder: 13},
	{Code: "job_run_status", Name: text("Job Run Status", "任务执行状态", "ジョブ実行ステータス", "작업 실행 상태"), IsSystem: true, IsEnabled: true, SortOrder: 14},
	{Code: "notification_email_status", Name: text("Notification Email Status", "通知邮件状态", "通知メールのステータス", "알림 이메일 상태"), IsSystem: true, IsEnabled: true, SortOrder: 16},
	{Code: "job_trigger", Name: text("Job Trigger", "任务触发方式", "ジョブの起動方法", "작업 실행 방식"), IsSystem: true, IsEnabled: true, SortOrder: 15},
}

// DefaultDictItems 默认字典项。种子项一律 IsSystem，Color 只能取 Colors 中的值，
// Icon 必须是前端 Icons 的 key（由 seed_integrity_test.go 与前端 lib/dict.test.ts 把关）。
var DefaultDictItems = []DictItem{
	{TypeCode: "gender", Label: text("Male", "男", "男性", "남성"), Value: "MALE", SortOrder: 1, IsSystem: true, IsEnabled: true},
	{TypeCode: "gender", Label: text("Female", "女", "女性", "여성"), Value: "FEMALE", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "gender", Label: text("Unknown", "未知", "不明", "알 수 없음"), Value: "UNKNOWN", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "status", Label: text("Enabled", "启用", "有効", "활성"), Value: "ENABLED", Color: "green", SortOrder: 1, IsSystem: true, IsEnabled: true, IsDefault: true},
	{TypeCode: "status", Label: text("Disabled", "禁用", "無効", "비활성"), Value: "DISABLED", Color: "red", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "status", Label: text("Pending", "待处理", "保留中", "대기 중"), Value: "PENDING", Color: "orange", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "priority", Label: text("Low", "低", "低", "낮음"), Value: "LOW", Color: "gray", SortOrder: 1, IsSystem: true, IsEnabled: true},
	{TypeCode: "priority", Label: text("Medium", "中", "中", "보통"), Value: "MEDIUM", Color: "blue", SortOrder: 2, IsSystem: true, IsEnabled: true, IsDefault: true},
	{TypeCode: "priority", Label: text("High", "高", "高", "높음"), Value: "HIGH", Color: "orange", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "priority", Label: text("Urgent", "紧急", "緊急", "긴급"), Value: "URGENT", Color: "red", SortOrder: 4, IsSystem: true, IsEnabled: true},
	{TypeCode: "boolean", Label: text("Yes", "是", "はい", "예"), Value: "true", SortOrder: 1, IsSystem: true, IsEnabled: true},
	{TypeCode: "boolean", Label: text("No", "否", "いいえ", "아니오"), Value: "false", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "asset_status", Label: text("Active", "生效", "有効", "활성"), Value: "ACTIVE", Color: "green", Icon: "check", SortOrder: 2, IsSystem: true, IsEnabled: true, IsDefault: true},
	{TypeCode: "asset_status", Label: text("Archived", "已归档", "アーカイブ済み", "보관됨"), Value: "ARCHIVED", Color: "gray", Icon: "archive", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "asset_category", Label: text("Image", "图片", "画像", "이미지"), Value: "IMAGE", Icon: "media", SortOrder: 1, IsSystem: true, IsEnabled: true},
	{TypeCode: "asset_category", Label: text("Video", "视频", "動画", "동영상"), Value: "VIDEO", Icon: "video", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "asset_category", Label: text("Audio", "音频", "音声", "오디오"), Value: "AUDIO", Icon: "music", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "asset_category", Label: text("Document", "文档", "ドキュメント", "문서"), Value: "DOCUMENT", Icon: "post", SortOrder: 4, IsSystem: true, IsEnabled: true},
	{TypeCode: "asset_category", Label: text("Archive", "压缩包", "アーカイブ", "압축 파일"), Value: "ARCHIVE", Icon: "archive", SortOrder: 5, IsSystem: true, IsEnabled: true},
	{TypeCode: "asset_category", Label: text("Other", "其他", "その他", "기타"), Value: "OTHER", Icon: "page", SortOrder: 6, IsSystem: true, IsEnabled: true, IsDefault: true},
	{TypeCode: "asset_scope", Label: text("Library", "资产库", "ライブラリ", "라이브러리"), Value: "LIBRARY", Color: "blue", Icon: "workspace", SortOrder: 1, IsSystem: true, IsEnabled: true, IsDefault: true},
	{TypeCode: "asset_scope", Label: text("Attachment", "业务附件", "業務添付", "업무 첨부"), Value: "ATTACHMENT", Color: "purple", Icon: "paperclip", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "account_source", Label: text("Local", "本地", "ローカル", "로컬"), Value: "INTERNAL", Icon: "mail", SortOrder: 1, IsSystem: true, IsEnabled: true, IsDefault: true},
	{TypeCode: "account_source", Label: text("Facebook", "Facebook", "Facebook", "Facebook"), Value: "FACEBOOK", Icon: "globe", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "account_source", Label: text("QQ", "QQ", "QQ", "QQ"), Value: "QQ", Icon: "globe", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "account_source", Label: text("WeChat", "微信", "WeChat", "위챗"), Value: "WECHAT", Icon: "globe", SortOrder: 4, IsSystem: true, IsEnabled: true},
	{TypeCode: "account_source", Label: text("Google", "Google", "Google", "Google"), Value: "GOOGLE", Icon: "globe", SortOrder: 5, IsSystem: true, IsEnabled: true},
	{TypeCode: "account_source", Label: text("GitHub", "GitHub", "GitHub", "GitHub"), Value: "GITHUB", Icon: "globe", SortOrder: 6, IsSystem: true, IsEnabled: true},
	{TypeCode: "account_source", Label: text("Single sign-on", "单点登录", "シングルサインオン", "SSO"), Value: "OIDC", Icon: "shield", SortOrder: 7, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_type", Label: text("System", "系统", "システム", "시스템"), Value: "SYSTEM", Color: "blue", Icon: "settings", SortOrder: 1, IsSystem: true, IsEnabled: true, IsDefault: true},
	{TypeCode: "notification_type", Label: text("Announcement", "公告", "お知らせ", "공지"), Value: "ANNOUNCE", Color: "purple", Icon: "speaker", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_type", Label: text("Message", "消息", "メッセージ", "메시지"), Value: "MESSAGE", Color: "gray", Icon: "mail", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_type", Label: text("Alert", "告警", "アラート", "경고"), Value: "ALERT", Color: "red", Icon: "warning", SortOrder: 4, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_type", Label: text("Task", "任务", "タスク", "작업"), Value: "TASK", Color: "blue", Icon: "kanban", SortOrder: 5, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_level", Label: text("Info", "信息", "情報", "정보"), Value: "INFO", Color: "blue", SortOrder: 1, IsSystem: true, IsEnabled: true, IsDefault: true},
	{TypeCode: "notification_level", Label: text("Success", "成功", "成功", "성공"), Value: "SUCCESS", Color: "green", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_level", Label: text("Warning", "警告", "警告", "경고"), Value: "WARNING", Color: "orange", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_level", Label: text("Error", "错误", "エラー", "오류"), Value: "ERROR", Color: "red", SortOrder: 4, IsSystem: true, IsEnabled: true},
	{TypeCode: "login_method", Label: text("Password", "密码", "パスワード", "비밀번호"), Value: "password", SortOrder: 1, IsSystem: true, IsEnabled: true, IsDefault: true},
	{TypeCode: "login_method", Label: text("Email OTP", "邮箱验证码", "メール OTP", "이메일 OTP"), Value: "email", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "login_method", Label: text("Mobile OTP", "短信验证码", "SMS OTP", "문자 OTP"), Value: "mobile", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "login_method", Label: text("Single sign-on", "单点登录", "シングルサインオン", "SSO"), Value: "oidc", SortOrder: 4, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Add Role", "新增角色", "ロール追加", "역할 추가"), Value: "ADD_ROLE", SortOrder: 1, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Delete Role", "删除角色", "ロール削除", "역할 삭제"), Value: "DELETE_ROLE", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Assign Role", "分配角色", "ロール割り当て", "역할 할당"), Value: "ADD_ROLE_FOR_USER", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Remove Role", "移除角色", "ロール解除", "역할 해제"), Value: "DELETE_ROLE_FOR_USER", SortOrder: 4, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Role Permissions Set", "设置角色权限", "ロール権限設定", "역할 권한 설정"), Value: "ROLE_PERMISSIONS_SET", SortOrder: 5, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Login", "登录", "ログイン", "로그인"), Value: "LOGIN", SortOrder: 7, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Logout", "登出", "ログアウト", "로그아웃"), Value: "LOGOUT", SortOrder: 8, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Password Change", "修改密码", "パスワード変更", "비밀번호 변경"), Value: "PASSWORD_CHANGE", SortOrder: 9, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Password Reset", "重置密码", "パスワードリセット", "비밀번호 재설정"), Value: "PASSWORD_RESET", SortOrder: 10, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("User Create", "创建用户", "ユーザー作成", "사용자 생성"), Value: "USER_CREATE", SortOrder: 11, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("User Update", "更新用户", "ユーザー更新", "사용자 수정"), Value: "USER_UPDATE", SortOrder: 12, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("User Delete", "删除用户", "ユーザー削除", "사용자 삭제"), Value: "USER_DELETE", SortOrder: 13, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Notification Create", "创建通知", "通知作成", "알림 생성"), Value: "NOTIFICATION_CREATE", SortOrder: 14, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Notification Update", "更新通知", "通知更新", "알림 수정"), Value: "NOTIFICATION_UPDATE", SortOrder: 15, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Notification Delete", "删除通知", "通知削除", "알림 삭제"), Value: "NOTIFICATION_DELETE", SortOrder: 16, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Asset Create", "创建资产", "アセット作成", "자산 생성"), Value: "ASSET_CREATE", SortOrder: 17, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Asset Update", "更新资产", "アセット更新", "자산 수정"), Value: "ASSET_UPDATE", SortOrder: 18, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Asset Delete", "删除资产", "アセット削除", "자산 삭제"), Value: "ASSET_DELETE", SortOrder: 19, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Setting Create", "创建配置", "設定作成", "설정 생성"), Value: "SETTING_CREATE", SortOrder: 20, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Setting Update", "更新配置", "設定更新", "설정 수정"), Value: "SETTING_UPDATE", SortOrder: 21, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Setting Delete", "删除配置", "設定削除", "설정 삭제"), Value: "SETTING_DELETE", SortOrder: 22, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Dict Type Create", "创建字典类型", "辞書タイプ作成", "사전 유형 생성"), Value: "DICT_TYPE_CREATE", SortOrder: 23, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Dict Type Update", "更新字典类型", "辞書タイプ更新", "사전 유형 수정"), Value: "DICT_TYPE_UPDATE", SortOrder: 24, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Dict Type Delete", "删除字典类型", "辞書タイプ削除", "사전 유형 삭제"), Value: "DICT_TYPE_DELETE", SortOrder: 25, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Dict Item Create", "创建字典项", "辞書項目作成", "사전 항목 생성"), Value: "DICT_ITEM_CREATE", SortOrder: 26, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Dict Item Update", "更新字典项", "辞書項目更新", "사전 항목 수정"), Value: "DICT_ITEM_UPDATE", SortOrder: 27, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Dict Item Delete", "删除字典项", "辞書項目削除", "사전 항목 삭제"), Value: "DICT_ITEM_DELETE", SortOrder: 28, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Profile Update", "更新个人资料", "プロフィール更新", "프로필 수정"), Value: "PROFILE_UPDATE", SortOrder: 29, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Avatar Update", "更新头像", "アバター更新", "아바타 수정"), Value: "AVATAR_UPDATE", SortOrder: 30, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Active Roles Change", "变更活动角色", "アクティブロール変更", "활성 역할 변경"), Value: "ACTIVE_ROLES_CHANGE", SortOrder: 31, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Audit Log Cleanup", "清理审计日志", "監査ログ整理", "감사 로그 정리"), Value: "AUDIT_LOG_CLEANUP", SortOrder: 32, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Login History Delete", "删除登录历史", "ログイン履歴削除", "로그인 기록 삭제"), Value: "LOGIN_HISTORY_DELETE", SortOrder: 33, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Update Role", "更新角色", "ロール更新", "역할 수정"), Value: "UPDATE_ROLE", SortOrder: 34, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Session Revoke", "强制下线", "強制サインアウト", "강제 로그아웃"), Value: "SESSION_REVOKE", SortOrder: 35, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Department Create", "新建部门", "部署作成", "부서 생성"), Value: "DEPT_CREATE", SortOrder: 36, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Department Update", "修改部门", "部署更新", "부서 수정"), Value: "DEPT_UPDATE", SortOrder: 37, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Department Delete", "删除部门", "部署削除", "부서 삭제"), Value: "DEPT_DELETE", SortOrder: 38, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Scheduled Job Update", "修改定时任务", "スケジュールジョブ更新", "예약 작업 수정"), Value: "JOB_UPDATE", SortOrder: 39, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Scheduled Job Run", "手动执行定时任务", "スケジュールジョブ手動実行", "예약 작업 수동 실행"), Value: "JOB_RUN", SortOrder: 40, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("User Export", "导出用户", "ユーザーエクスポート", "사용자 내보내기"), Value: "USER_EXPORT", SortOrder: 41, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("User Import", "导入用户", "ユーザーインポート", "사용자 가져오기"), Value: "USER_IMPORT", SortOrder: 42, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Audit Log Export", "导出审计日志", "監査ログエクスポート", "감사 로그 내보내기"), Value: "AUDIT_LOG_EXPORT", SortOrder: 43, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Login History Export", "导出登录历史", "ログイン履歴エクスポート", "로그인 기록 내보내기"), Value: "LOGIN_HISTORY_EXPORT", SortOrder: 44, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Two-factor Setup Started", "开始绑定两步验证", "2 段階認証の設定開始", "2단계 인증 설정 시작"), Value: "MFA_SETUP", SortOrder: 45, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Two-factor Enabled", "开启两步验证", "2 段階認証を有効化", "2단계 인증 켜기"), Value: "MFA_ENABLE", SortOrder: 46, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Two-factor Disabled", "关闭两步验证", "2 段階認証を無効化", "2단계 인증 끄기"), Value: "MFA_DISABLE", SortOrder: 47, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Recovery Codes Regenerated", "重新生成恢复码", "リカバリーコード再生成", "복구 코드 재생성"), Value: "MFA_RECOVERY_CODES", SortOrder: 48, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Two-factor Reset", "重置两步验证", "2 段階認証リセット", "2단계 인증 초기화"), Value: "MFA_RESET", SortOrder: 49, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Account Linked", "关联外部账号", "外部アカウント連携", "외부 계정 연결"), Value: "IDENTITY_LINK", SortOrder: 50, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Account Unlinked", "解除外部账号关联", "外部アカウント連携解除", "외부 계정 연결 해제"), Value: "IDENTITY_UNLINK", SortOrder: 51, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Identity Provider Create", "新建身份提供方", "ID プロバイダー作成", "ID 공급자 생성"), Value: "OIDC_PROVIDER_CREATE", SortOrder: 52, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Identity Provider Update", "修改身份提供方", "ID プロバイダー更新", "ID 공급자 수정"), Value: "OIDC_PROVIDER_UPDATE", SortOrder: 53, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Identity Provider Delete", "删除身份提供方", "ID プロバイダー削除", "ID 공급자 삭제"), Value: "OIDC_PROVIDER_DELETE", SortOrder: 54, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Resource Create", "新建资源", "リソース作成", "리소스 생성"), Value: "RESOURCE_CREATE", SortOrder: 55, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Resource Update", "修改资源", "リソース更新", "리소스 수정"), Value: "RESOURCE_UPDATE", SortOrder: 56, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Resource Delete", "删除资源", "リソース削除", "리소스 삭제"), Value: "RESOURCE_DELETE", SortOrder: 57, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Menu Create", "新建菜单", "メニュー作成", "메뉴 생성"), Value: "MENU_CREATE", SortOrder: 58, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Menu Update", "修改菜单", "メニュー更新", "메뉴 수정"), Value: "MENU_UPDATE", SortOrder: 59, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("Menu Delete", "删除菜单", "メニュー削除", "메뉴 삭제"), Value: "MENU_DELETE", SortOrder: 60, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("SoD Constraint Create", "新建职责分离约束", "職務分離制約作成", "직무 분리 제약 생성"), Value: "SOD_CONSTRAINT_CREATE", SortOrder: 61, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("SoD Constraint Update", "修改职责分离约束", "職務分離制約更新", "직무 분리 제약 수정"), Value: "SOD_CONSTRAINT_UPDATE", SortOrder: 62, IsSystem: true, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: text("SoD Constraint Delete", "删除职责分离约束", "職務分離制約削除", "직무 분리 제약 삭제"), Value: "SOD_CONSTRAINT_DELETE", SortOrder: 63, IsSystem: true, IsEnabled: true},
	// 取值与 permission.DataScope 常量一一对应。
	{TypeCode: "role_data_scope", Label: text("All data", "全部数据", "全データ", "전체 데이터"), Value: "ALL", Color: "red", SortOrder: 1, IsSystem: true, IsEnabled: true},
	{TypeCode: "role_data_scope", Label: text("Own department and below", "本部门及以下", "所属部署と配下", "소속 부서 및 하위"), Value: "DEPT_AND_CHILDREN", Color: "orange", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "role_data_scope", Label: text("Own department", "本部门", "所属部署のみ", "소속 부서"), Value: "DEPT", Color: "blue", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "role_data_scope", Label: text("Selected departments", "指定部门", "指定部署", "지정 부서"), Value: "CUSTOM", Color: "purple", SortOrder: 4, IsSystem: true, IsEnabled: true},
	{TypeCode: "role_data_scope", Label: text("Only themselves", "仅本人", "本人のみ", "본인만"), Value: "SELF", Color: "gray", SortOrder: 5, IsSystem: true, IsEnabled: true},
	// 取值与 notification.EmailStatus 常量一一对应。
	{TypeCode: "notification_email_status", Label: text("Pending", "待发送", "送信待ち", "대기 중"), Value: "PENDING", Color: "blue", SortOrder: 1, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_email_status", Label: text("Sent", "已发送", "送信済み", "보냄"), Value: "SENT", Color: "green", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_email_status", Label: text("Failed", "发送失败", "送信失敗", "실패"), Value: "FAILED", Color: "red", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "notification_email_status", Label: text("Skipped", "未发送", "送信せず", "건너뜀"), Value: "SKIPPED", Color: "gray", SortOrder: 4, IsSystem: true, IsEnabled: true},
	// 取值与 job.Status / job.Trigger 常量一一对应。
	{TypeCode: "job_run_status", Label: text("Running", "执行中", "実行中", "실행 중"), Value: "RUNNING", Color: "blue", SortOrder: 1, IsSystem: true, IsEnabled: true},
	{TypeCode: "job_run_status", Label: text("Succeeded", "成功", "成功", "성공"), Value: "SUCCEEDED", Color: "green", SortOrder: 2, IsSystem: true, IsEnabled: true},
	{TypeCode: "job_run_status", Label: text("Failed", "失败", "失敗", "실패"), Value: "FAILED", Color: "red", SortOrder: 3, IsSystem: true, IsEnabled: true},
	{TypeCode: "job_trigger", Label: text("Scheduled", "按计划", "スケジュール", "예약 실행"), Value: "SCHEDULE", Color: "gray", SortOrder: 1, IsSystem: true, IsEnabled: true},
	{TypeCode: "job_trigger", Label: text("Manual", "手动", "手動", "수동"), Value: "MANUAL", Color: "purple", SortOrder: 2, IsSystem: true, IsEnabled: true},
}
