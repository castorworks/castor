// Package models 定义 GORM 持久化模型。
// 这些模型包含 GORM 标签和生命周期钩子，与 Domain 层的纯业务实体分离。
// Domain 层实体不依赖 GORM，Repository 接口使用 Domain 实体，
// Repository 实现内部使用本包模型进行 ORM 操作。
package models

import (
	"time"

	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"gorm.io/gorm"
)

// ============================================
// 审计字段工具
// ============================================

// ExtractUserIDFromContext 从 gorm.DB 的 context 中提取用户 ID
func ExtractUserIDFromContext(tx *gorm.DB) (uint, uint) {
	if ctx := tx.Statement.Context; ctx != nil {
		userID := ucontext.AuditUserIDFromContext(ctx)
		if userID != 0 {
			return userID, userID
		}
	}
	return 0, 0
}

// ============================================
// User
// ============================================

type UserModel struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	CreatedBy uint `gorm:"index:idx_users_created_by"`
	UpdatedAt time.Time
	UpdatedBy uint `gorm:"index:idx_users_updated_by"`

	Username             string `gorm:"type:varchar(128);uniqueIndex:idx_username_account_source"`
	AccountSource        string `gorm:"type:varchar(128);uniqueIndex:idx_username_account_source"`
	Name                 string
	Avatar               string
	Password             string
	Enable               bool
	Locked               bool
	AccountExpireDate    time.Time
	CredentialExpireDate time.Time
}

func (UserModel) TableName() string { return "users" }

func (m *UserModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *UserModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *UserModel) ToEntity() *user.User {
	return &user.User{
		ID:                   m.ID,
		CreatedAt:            m.CreatedAt,
		CreatedBy:            m.CreatedBy,
		UpdatedAt:            m.UpdatedAt,
		UpdatedBy:            m.UpdatedBy,
		Username:             m.Username,
		AccountSource:        m.AccountSource,
		Name:                 m.Name,
		Avatar:               m.Avatar,
		Password:             m.Password,
		Enable:               m.Enable,
		Locked:               m.Locked,
		AccountExpireDate:    m.AccountExpireDate,
		CredentialExpireDate: m.CredentialExpireDate,
	}
}

func UserModelFromEntity(e *user.User) *UserModel {
	return &UserModel{
		ID:                   e.ID,
		CreatedAt:            e.CreatedAt,
		CreatedBy:            e.CreatedBy,
		UpdatedAt:            e.UpdatedAt,
		UpdatedBy:            e.UpdatedBy,
		Username:             e.Username,
		AccountSource:        e.AccountSource,
		Name:                 e.Name,
		Avatar:               e.Avatar,
		Password:             e.Password,
		Enable:               e.Enable,
		Locked:               e.Locked,
		AccountExpireDate:    e.AccountExpireDate,
		CredentialExpireDate: e.CredentialExpireDate,
	}
}

// ============================================
// Asset
// ============================================

type AssetModel struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	CreatedBy uint `gorm:"index"`
	UpdatedAt time.Time
	UpdatedBy uint `gorm:"index"`

	Name          string            `gorm:"type:varchar(255)"`
	Filename      string            `gorm:"type:varchar(255);not null"`
	Description   string            `gorm:"type:text"`
	Tags          string            `gorm:"type:varchar(500)"`
	FolderID      *uint             `gorm:"index:idx_assets_folder"`
	FolderPath    string            `gorm:"type:varchar(500);index:idx_assets_path"`
	ObjectKey     string            `gorm:"type:varchar(500);uniqueIndex:idx_assets_object_key"`
	StorageType   asset.StorageType `gorm:"type:varchar(32);default:'S3'"`
	Bucket        string            `gorm:"type:varchar(128)"`
	Extension     string            `gorm:"type:varchar(32);index:idx_assets_extension"`
	MimeType      string            `gorm:"type:varchar(128);index:idx_assets_mime_type"`
	Size          int64
	Hash          string              `gorm:"type:varchar(64);index:idx_assets_hash"`
	Category      asset.AssetCategory `gorm:"type:varchar(32);index:idx_assets_category;default:'OTHER'"`
	Status        asset.AssetStatus   `gorm:"type:varchar(32);index:idx_assets_status;default:'ACTIVE'"`
	IsPublic      bool                `gorm:"default:false;index:idx_assets_public"`
	Width         int
	Height        int
	Duration      int
	ThumbnailKey  string `gorm:"type:varchar(500)"`
	DownloadCount int    `gorm:"default:0"`
	ViewCount     int    `gorm:"default:0"`
}

func (AssetModel) TableName() string { return "assets" }

func (m *AssetModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *AssetModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m AssetModel) ToEntity() *asset.Asset {
	e := &asset.Asset{
		Name:          m.Name,
		Filename:      m.Filename,
		Description:   m.Description,
		Tags:          m.Tags,
		FolderID:      m.FolderID,
		FolderPath:    m.FolderPath,
		ObjectKey:     m.ObjectKey,
		StorageType:   m.StorageType,
		Bucket:        m.Bucket,
		Extension:     m.Extension,
		MimeType:      m.MimeType,
		Size:          m.Size,
		Hash:          m.Hash,
		Category:      m.Category,
		Status:        m.Status,
		IsPublic:      m.IsPublic,
		Width:         m.Width,
		Height:        m.Height,
		Duration:      m.Duration,
		ThumbnailKey:  m.ThumbnailKey,
		DownloadCount: m.DownloadCount,
		ViewCount:     m.ViewCount,
	}
	e.ID = m.ID
	e.CreatedAt = m.CreatedAt
	e.CreatedBy = m.CreatedBy
	e.UpdatedAt = m.UpdatedAt
	e.UpdatedBy = m.UpdatedBy
	return e
}

func AssetModelFromEntity(e *asset.Asset) *AssetModel {
	return &AssetModel{
		ID:            e.BaseModel.ID,
		CreatedAt:     e.BaseModel.CreatedAt,
		CreatedBy:     e.BaseModel.CreatedBy,
		UpdatedAt:     e.BaseModel.UpdatedAt,
		UpdatedBy:     e.BaseModel.UpdatedBy,
		Name:          e.Name,
		Filename:      e.Filename,
		Description:   e.Description,
		Tags:          e.Tags,
		FolderID:      e.FolderID,
		FolderPath:    e.FolderPath,
		ObjectKey:     e.ObjectKey,
		StorageType:   e.StorageType,
		Bucket:        e.Bucket,
		Extension:     e.Extension,
		MimeType:      e.MimeType,
		Size:          e.Size,
		Hash:          e.Hash,
		Category:      e.Category,
		Status:        e.Status,
		IsPublic:      e.IsPublic,
		Width:         e.Width,
		Height:        e.Height,
		Duration:      e.Duration,
		ThumbnailKey:  e.ThumbnailKey,
		DownloadCount: e.DownloadCount,
		ViewCount:     e.ViewCount,
	}
}

// ============================================
// AuditLog
// ============================================

type AuditLogModel struct {
	ID         uint `gorm:"primarykey"`
	CreatedAt  time.Time
	LogType    audit_log.AuditLogType `gorm:"type:varchar(50);index:idx_audit_log_type"`
	Operator   string                 `gorm:"type:varchar(128);index:idx_audit_operator"`
	OperatorID uint                   `gorm:"index:idx_audit_operator_id"`
	Target     string                 `gorm:"type:varchar(256)"`
	Details    string                 `gorm:"type:text"`
	IpAddr     string                 `gorm:"type:varchar(64)"`
	Success    bool                   `gorm:"index:idx_audit_success"`
}

func (AuditLogModel) TableName() string { return "audit_logs" }

func (m *AuditLogModel) ToEntity() *audit_log.AuditLog {
	return &audit_log.AuditLog{
		ID:         m.ID,
		CreatedAt:  m.CreatedAt,
		LogType:    m.LogType,
		Operator:   m.Operator,
		OperatorID: m.OperatorID,
		Target:     m.Target,
		Details:    m.Details,
		IpAddr:     m.IpAddr,
		Success:    m.Success,
	}
}

func AuditLogModelFromEntity(e *audit_log.AuditLog) *AuditLogModel {
	return &AuditLogModel{
		ID:         e.ID,
		CreatedAt:  e.CreatedAt,
		LogType:    e.LogType,
		Operator:   e.Operator,
		OperatorID: e.OperatorID,
		Target:     e.Target,
		Details:    e.Details,
		IpAddr:     e.IpAddr,
		Success:    e.Success,
	}
}

// ============================================
// Dictionary
// ============================================

type DictTypeModel struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	CreatedBy   uint `gorm:"index"`
	UpdatedAt   time.Time
	UpdatedBy   uint   `gorm:"index"`
	Code        string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name        string `gorm:"type:varchar(100);not null"`
	Description string `gorm:"type:varchar(500)"`
	IsSystem    bool   `gorm:"default:false"`
	IsEnabled   bool   `gorm:"default:true"`
	SortOrder   int    `gorm:"default:0"`
}

func (DictTypeModel) TableName() string { return "dict_types" }

func (m *DictTypeModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}
func (m *DictTypeModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *DictTypeModel) ToEntity() *dictionary.DictType {
	return &dictionary.DictType{
		ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy,
		UpdatedAt: m.UpdatedAt, UpdatedBy: m.UpdatedBy,
		Code: m.Code, Name: m.Name, Description: m.Description,
		IsSystem: m.IsSystem, IsEnabled: m.IsEnabled, SortOrder: m.SortOrder,
	}
}

type DictItemModel struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	CreatedBy   uint `gorm:"index"`
	UpdatedAt   time.Time
	UpdatedBy   uint   `gorm:"index"`
	TypeCode    string `gorm:"type:varchar(100);index:idx_dict_type_code;not null"`
	Label       string `gorm:"type:varchar(200);not null"`
	Value       string `gorm:"type:varchar(200);not null"`
	Description string `gorm:"type:varchar(500)"`
	Extra       string `gorm:"type:text"`
	Color       string `gorm:"type:varchar(50)"`
	Icon        string `gorm:"type:varchar(100)"`
	ParentID    *uint  `gorm:"index"`
	IsDefault   bool   `gorm:"default:false"`
	IsEnabled   bool   `gorm:"default:true"`
	SortOrder   int    `gorm:"default:0"`
}

func (DictItemModel) TableName() string { return "dict_items" }

func (m *DictItemModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}
func (m *DictItemModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *DictItemModel) ToEntity() *dictionary.DictItem {
	return &dictionary.DictItem{
		ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy,
		UpdatedAt: m.UpdatedAt, UpdatedBy: m.UpdatedBy,
		TypeCode: m.TypeCode, Label: m.Label, Value: m.Value,
		Description: m.Description, Extra: m.Extra, Color: m.Color,
		Icon: m.Icon, ParentID: m.ParentID, IsDefault: m.IsDefault,
		IsEnabled: m.IsEnabled, SortOrder: m.SortOrder,
	}
}

// ============================================
// LoginHistory
// ============================================

type LoginHistoryModel struct {
	ID          uint      `gorm:"primarykey"`
	CreatedAt   time.Time `gorm:"index:idx_login_history_created_at"`
	CreatedBy   uint      `gorm:"index"`
	UpdatedAt   time.Time
	UpdatedBy   uint   `gorm:"index"`
	UserID      uint   `gorm:"index:idx_login_history_user_id"`
	Username    string `gorm:"type:varchar(128);index:idx_login_history_username"`
	IpAddr      string `gorm:"type:varchar(45)"`
	UserAgent   string `gorm:"type:varchar(512)"`
	LoginMethod string `gorm:"type:varchar(32)"`
	Success     bool   `gorm:"index:idx_login_history_success"`
}

func (LoginHistoryModel) TableName() string { return "login_histories" }

func (m *LoginHistoryModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *LoginHistoryModel) ToEntity() *login_history.LoginHistory {
	e := &login_history.LoginHistory{
		UserID:      m.UserID,
		Username:    m.Username,
		IpAddr:      m.IpAddr,
		UserAgent:   m.UserAgent,
		LoginMethod: m.LoginMethod,
		Success:     m.Success,
	}
	e.ID = m.ID
	e.CreatedAt = m.CreatedAt
	e.CreatedBy = m.CreatedBy
	e.UpdatedAt = m.UpdatedAt
	e.UpdatedBy = m.UpdatedBy
	return e
}

func LoginHistoryModelFromEntity(e *login_history.LoginHistory) *LoginHistoryModel {
	return &LoginHistoryModel{
		ID:          e.BaseModel.ID,
		CreatedAt:   e.BaseModel.CreatedAt,
		CreatedBy:   e.BaseModel.CreatedBy,
		UpdatedAt:   e.BaseModel.UpdatedAt,
		UpdatedBy:   e.BaseModel.UpdatedBy,
		UserID:      e.UserID,
		Username:    e.Username,
		IpAddr:      e.IpAddr,
		UserAgent:   e.UserAgent,
		LoginMethod: e.LoginMethod,
		Success:     e.Success,
	}
}

// ============================================
// Notification
// ============================================

type NotificationModel struct {
	ID           uint `gorm:"primarykey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    uint
	UpdatedBy    uint
	Title        string                         `gorm:"type:varchar(200);not null"`
	Content      string                         `gorm:"type:text"`
	TemplateKey  string                         `gorm:"type:varchar(128);index"`
	TemplateData string                         `gorm:"type:text"`
	Type         notification.NotificationType  `gorm:"type:varchar(20);index;default:SYSTEM"`
	Level        notification.NotificationLevel `gorm:"type:varchar(20);default:INFO"`
	Link         string                         `gorm:"type:varchar(500)"`
	Extra        string                         `gorm:"type:text"`
	SenderID     uint                           `gorm:"index"`
	IsGlobal     bool                           `gorm:"default:false"`
	ExpireAt     *time.Time
}

func (NotificationModel) TableName() string { return "notifications" }

func (m *NotificationModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, _ = ExtractUserIDFromContext(tx)
	return nil
}
func (m *NotificationModel) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedBy, _ = ExtractUserIDFromContext(tx)
	return nil
}

func (m *NotificationModel) ToEntity() *notification.Notification {
	return &notification.Notification{
		ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		CreatedBy: m.CreatedBy, UpdatedBy: m.UpdatedBy,
		Title: m.Title, Content: m.Content, TemplateKey: m.TemplateKey,
		TemplateData: m.TemplateData, Type: m.Type, Level: m.Level,
		Link: m.Link, Extra: m.Extra, SenderID: m.SenderID,
		IsGlobal: m.IsGlobal, ExpireAt: m.ExpireAt,
	}
}

type UserNotificationModel struct {
	ID             uint `gorm:"primarykey"`
	CreatedAt      time.Time
	UserID         uint `gorm:"uniqueIndex:idx_user_notification;not null"`
	NotificationID uint `gorm:"uniqueIndex:idx_user_notification;not null"`
	IsRead         bool `gorm:"default:false;index"`
	ReadAt         *time.Time
	IsDismissed    bool `gorm:"default:false;index"`
	DismissedAt    *time.Time
}

func (UserNotificationModel) TableName() string { return "user_notifications" }

// ============================================
// Permission
// ============================================

type ResourceModel struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	CreatedBy   uint `gorm:"index"`
	UpdatedAt   time.Time
	UpdatedBy   uint                        `gorm:"index"`
	Code        string                      `gorm:"uniqueIndex;size:100;not null"`
	Name        string                      `gorm:"size:100;not null"`
	Description string                      `gorm:"size:500"`
	Path        string                      `gorm:"size:200;not null;index"`
	Actions     permission.StringSlice      `gorm:"type:text;not null"`
	Category    permission.ResourceCategory `gorm:"size:50;not null;index;check:chk_rbac_resource_category,category IN ('admin','user')"`
	Module      string                      `gorm:"size:100;index"`
	SortOrder   int                         `gorm:"default:0"`
	IsSystem    bool                        `gorm:"default:false"`
	IsEnabled   bool                        `gorm:"default:true"`
}

func (ResourceModel) TableName() string { return "rbac_resources" }

func (m *ResourceModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}
func (m *ResourceModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *ResourceModel) ToEntity() *permission.Resource {
	return &permission.Resource{
		ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy,
		UpdatedAt: m.UpdatedAt, UpdatedBy: m.UpdatedBy,
		Code: m.Code, Name: m.Name, Description: m.Description,
		Path: m.Path, Actions: m.Actions, Category: m.Category,
		Module: m.Module, SortOrder: m.SortOrder,
		IsSystem: m.IsSystem, IsEnabled: m.IsEnabled,
	}
}

type RoleModel struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	CreatedBy   uint `gorm:"index"`
	UpdatedAt   time.Time
	UpdatedBy   uint   `gorm:"index"`
	Code        string `gorm:"uniqueIndex;size:50;not null"`
	Name        string `gorm:"size:100;not null"`
	Description string `gorm:"size:500"`
	IsSystem    bool   `gorm:"default:false"`
	IsEnabled   bool   `gorm:"default:true"`
}

func (RoleModel) TableName() string { return "rbac_roles" }

func (m *RoleModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}
func (m *RoleModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *RoleModel) ToEntity() *permission.Role {
	return &permission.Role{
		ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy,
		UpdatedAt: m.UpdatedAt, UpdatedBy: m.UpdatedBy,
		Code: m.Code, Name: m.Name, Description: m.Description,
		IsSystem: m.IsSystem, IsEnabled: m.IsEnabled,
	}
}

type RolePermissionModel struct {
	ID         uint `gorm:"primarykey"`
	CreatedAt  time.Time
	CreatedBy  uint          `gorm:"index"`
	RoleID     uint          `gorm:"not null;uniqueIndex:idx_rbac_role_permission"`
	ResourceID uint          `gorm:"not null;uniqueIndex:idx_rbac_role_permission"`
	Action     string        `gorm:"size:16;not null;uniqueIndex:idx_rbac_role_permission;check:chk_rbac_role_permission_action,action IN ('GET','POST','PUT','PATCH','DELETE')"`
	IsSystem   bool          `gorm:"default:false"`
	Role       RoleModel     `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Resource   ResourceModel `gorm:"foreignKey:ResourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (RolePermissionModel) TableName() string { return "rbac_role_permissions" }

func (m *RolePermissionModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, _ = ExtractUserIDFromContext(tx)
	return nil
}

type UserRoleModel struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	CreatedBy uint      `gorm:"index"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_rbac_user_role"`
	RoleID    uint      `gorm:"not null;uniqueIndex:idx_rbac_user_role"`
	IsSystem  bool      `gorm:"default:false"`
	User      UserModel `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Role      RoleModel `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (UserRoleModel) TableName() string { return "rbac_user_roles" }

func (m *UserRoleModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, _ = ExtractUserIDFromContext(tx)
	return nil
}

type RoleHierarchyModel struct {
	ID           uint `gorm:"primarykey"`
	CreatedAt    time.Time
	CreatedBy    uint      `gorm:"index"`
	SeniorRoleID uint      `gorm:"not null;uniqueIndex:idx_rbac_role_hierarchy;check:chk_rbac_role_hierarchy_not_self,senior_role_id <> junior_role_id"`
	JuniorRoleID uint      `gorm:"not null;uniqueIndex:idx_rbac_role_hierarchy"`
	SeniorRole   RoleModel `gorm:"foreignKey:SeniorRoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	JuniorRole   RoleModel `gorm:"foreignKey:JuniorRoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (RoleHierarchyModel) TableName() string { return "rbac_role_hierarchies" }

func (m *RoleHierarchyModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, _ = ExtractUserIDFromContext(tx)
	return nil
}

type SeparationConstraintModel struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	CreatedBy   uint `gorm:"index"`
	UpdatedAt   time.Time
	UpdatedBy   uint                      `gorm:"index"`
	Code        string                    `gorm:"size:64;not null;uniqueIndex"`
	Name        string                    `gorm:"size:128;not null"`
	Description string                    `gorm:"size:500"`
	Type        permission.ConstraintType `gorm:"size:3;not null;index;check:chk_rbac_constraint_type,type IN ('SSD','DSD')"`
	Cardinality int                       `gorm:"not null;check:chk_rbac_constraint_cardinality,cardinality >= 2"`
	IsEnabled   bool                      `gorm:"default:true;index"`
}

func (SeparationConstraintModel) TableName() string { return "rbac_constraints" }

func (m *SeparationConstraintModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *SeparationConstraintModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

type ConstraintRoleModel struct {
	ConstraintID uint                      `gorm:"primaryKey;autoIncrement:false"`
	RoleID       uint                      `gorm:"primaryKey;autoIncrement:false"`
	Constraint   SeparationConstraintModel `gorm:"foreignKey:ConstraintID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Role         RoleModel                 `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (ConstraintRoleModel) TableName() string { return "rbac_constraint_roles" }

type AuthorizationSessionModel struct {
	ID        string     `gorm:"type:char(36);primaryKey"`
	UserID    uint       `gorm:"not null;index"`
	CreatedAt time.Time  `gorm:"not null"`
	ExpiresAt time.Time  `gorm:"not null;index;check:chk_rbac_session_expiry,expires_at > created_at"`
	RevokedAt *time.Time `gorm:"index"`
	Version   uint64     `gorm:"not null;default:1;check:chk_rbac_session_version,version >= 1"`
	User      UserModel  `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (AuthorizationSessionModel) TableName() string { return "rbac_sessions" }

type SessionRoleModel struct {
	SessionID string                    `gorm:"type:char(36);primaryKey"`
	RoleID    uint                      `gorm:"primaryKey;autoIncrement:false"`
	Session   AuthorizationSessionModel `gorm:"foreignKey:SessionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Role      RoleModel                 `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (SessionRoleModel) TableName() string { return "rbac_session_roles" }

// ============================================
// Setting
// ============================================

type SettingModel struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	CreatedBy   uint `gorm:"index"`
	UpdatedAt   time.Time
	UpdatedBy   uint                    `gorm:"index"`
	Key         string                  `gorm:"type:varchar(100);uniqueIndex;not null"`
	Value       string                  `gorm:"type:text"`
	Type        setting.SettingType     `gorm:"type:varchar(20);default:STRING"`
	Category    setting.SettingCategory `gorm:"type:varchar(50);index;default:GENERAL"`
	Name        string                  `gorm:"type:varchar(100)"`
	Description string                  `gorm:"type:varchar(500)"`
	DefaultVal  string                  `gorm:"type:text"`
	Options     setting.SettingOptions  `gorm:"type:text"`
	IsSystem    bool                    `gorm:"default:false"`
	IsPublic    bool                    `gorm:"default:false"`
	SortOrder   int                     `gorm:"default:0"`
}

func (SettingModel) TableName() string { return "settings" }

func (m *SettingModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}
func (m *SettingModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *SettingModel) ToEntity() *setting.Setting {
	return &setting.Setting{
		ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy,
		UpdatedAt: m.UpdatedAt, UpdatedBy: m.UpdatedBy,
		Key: m.Key, Value: m.Value, Type: m.Type, Category: m.Category,
		Name: m.Name, Description: m.Description, DefaultVal: m.DefaultVal,
		Options: m.Options, IsSystem: m.IsSystem, IsPublic: m.IsPublic,
		SortOrder: m.SortOrder,
	}
}

func SettingModelFromEntity(e *setting.Setting) *SettingModel {
	return &SettingModel{
		ID: e.ID, CreatedAt: e.CreatedAt, CreatedBy: e.CreatedBy,
		UpdatedAt: e.UpdatedAt, UpdatedBy: e.UpdatedBy,
		Key: e.Key, Value: e.Value, Type: e.Type, Category: e.Category,
		Name: e.Name, Description: e.Description, DefaultVal: e.DefaultVal,
		Options: e.Options, IsSystem: e.IsSystem, IsPublic: e.IsPublic,
		SortOrder: e.SortOrder,
	}
}

func DictTypeModelFromEntity(e *dictionary.DictType) *DictTypeModel {
	return &DictTypeModel{
		ID: e.ID, CreatedAt: e.CreatedAt, CreatedBy: e.CreatedBy,
		UpdatedAt: e.UpdatedAt, UpdatedBy: e.UpdatedBy,
		Code: e.Code, Name: e.Name, Description: e.Description,
		IsSystem: e.IsSystem, IsEnabled: e.IsEnabled, SortOrder: e.SortOrder,
	}
}

func DictItemModelFromEntity(e *dictionary.DictItem) *DictItemModel {
	return &DictItemModel{
		ID: e.ID, CreatedAt: e.CreatedAt, CreatedBy: e.CreatedBy,
		UpdatedAt: e.UpdatedAt, UpdatedBy: e.UpdatedBy,
		TypeCode: e.TypeCode, Label: e.Label, Value: e.Value,
		Description: e.Description, Extra: e.Extra, Color: e.Color,
		Icon: e.Icon, ParentID: e.ParentID, IsDefault: e.IsDefault,
		IsEnabled: e.IsEnabled, SortOrder: e.SortOrder,
	}
}

func NotificationModelFromEntity(e *notification.Notification) *NotificationModel {
	return &NotificationModel{
		ID: e.ID, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
		CreatedBy: e.CreatedBy, UpdatedBy: e.UpdatedBy,
		Title: e.Title, Content: e.Content, TemplateKey: e.TemplateKey,
		TemplateData: e.TemplateData, Type: e.Type, Level: e.Level,
		Link: e.Link, Extra: e.Extra, SenderID: e.SenderID,
		IsGlobal: e.IsGlobal, ExpireAt: e.ExpireAt,
	}
}

func ResourceModelFromEntity(e *permission.Resource) *ResourceModel {
	return &ResourceModel{
		ID: e.ID, CreatedAt: e.CreatedAt, CreatedBy: e.CreatedBy,
		UpdatedAt: e.UpdatedAt, UpdatedBy: e.UpdatedBy,
		Code: e.Code, Name: e.Name, Description: e.Description,
		Path: e.Path, Actions: e.Actions, Category: e.Category,
		Module: e.Module, SortOrder: e.SortOrder,
		IsSystem: e.IsSystem, IsEnabled: e.IsEnabled,
	}
}

func RoleModelFromEntity(e *permission.Role) *RoleModel {
	return &RoleModel{
		ID: e.ID, CreatedAt: e.CreatedAt, CreatedBy: e.CreatedBy,
		UpdatedAt: e.UpdatedAt, UpdatedBy: e.UpdatedBy,
		Code: e.Code, Name: e.Name, Description: e.Description,
		IsSystem: e.IsSystem, IsEnabled: e.IsEnabled,
	}
}

func RolePermissionModelFromEntity(e *permission.RolePermission) *RolePermissionModel {
	return &RolePermissionModel{
		ID: e.ID, CreatedAt: e.CreatedAt, CreatedBy: e.CreatedBy,
		RoleID: e.RoleID, ResourceID: e.ResourceID,
		Action: e.Action, IsSystem: e.IsSystem,
	}
}
