// Package models 定义 GORM 持久化模型。
// 这些模型包含 GORM 标签和生命周期钩子，与 Domain 层的纯业务实体分离。
// Domain 层实体不依赖 GORM，Repository 接口使用 Domain 实体，
// Repository 实现内部使用本包模型进行 ORM 操作。
package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/castorworks/castor/internal/domain/job"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/domain/mfa"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/sso"
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

	Username      string `gorm:"type:varchar(128);uniqueIndex:idx_username_account_source"`
	AccountSource string `gorm:"type:varchar(128);uniqueIndex:idx_username_account_source"`
	// Email/Mobile 允许为空且可重复为空，因此唯一性由初始结构建立的
	// 部分唯一索引（WHERE <> ''）保证，这里只声明普通索引用于查找。
	Email                string `gorm:"type:varchar(255);index:idx_users_email"`
	Mobile               string `gorm:"type:varchar(32);index:idx_users_mobile"`
	EmailVerified        bool   `gorm:"not null;default:false"`
	MobileVerified       bool   `gorm:"not null;default:false"`
	Name                 string
	Avatar               string
	Password             string
	Enable               bool
	Locked               bool
	AccountExpireDate    time.Time
	CredentialExpireDate time.Time
	// DepartmentID 的外键（ON DELETE RESTRICT）由初始结构创建：
	// 这里不声明关联字段，避免用户的保存路径意外级联写部门。
	DepartmentID *uint `gorm:"index"`
	// MuteNotificationEmails 用户关闭了通知邮件
	MuteNotificationEmails bool `gorm:"not null;default:false"`
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
		ID:                     m.ID,
		CreatedAt:              m.CreatedAt,
		CreatedBy:              m.CreatedBy,
		UpdatedAt:              m.UpdatedAt,
		UpdatedBy:              m.UpdatedBy,
		Username:               m.Username,
		AccountSource:          m.AccountSource,
		Email:                  m.Email,
		Mobile:                 m.Mobile,
		EmailVerified:          m.EmailVerified,
		MobileVerified:         m.MobileVerified,
		Name:                   m.Name,
		Avatar:                 m.Avatar,
		Password:               m.Password,
		Enable:                 m.Enable,
		Locked:                 m.Locked,
		AccountExpireDate:      m.AccountExpireDate,
		CredentialExpireDate:   m.CredentialExpireDate,
		DepartmentID:           m.DepartmentID,
		MuteNotificationEmails: m.MuteNotificationEmails,
	}
}

func UserModelFromEntity(e *user.User) *UserModel {
	return &UserModel{
		ID:                     e.ID,
		CreatedAt:              e.CreatedAt,
		CreatedBy:              e.CreatedBy,
		UpdatedAt:              e.UpdatedAt,
		UpdatedBy:              e.UpdatedBy,
		Username:               e.Username,
		AccountSource:          e.AccountSource,
		Email:                  e.Email,
		Mobile:                 e.Mobile,
		EmailVerified:          e.EmailVerified,
		MobileVerified:         e.MobileVerified,
		Name:                   e.Name,
		Avatar:                 e.Avatar,
		Password:               e.Password,
		Enable:                 e.Enable,
		Locked:                 e.Locked,
		AccountExpireDate:      e.AccountExpireDate,
		CredentialExpireDate:   e.CredentialExpireDate,
		DepartmentID:           e.DepartmentID,
		MuteNotificationEmails: e.MuteNotificationEmails,
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

	Name        string `gorm:"type:varchar(255)"`
	Filename    string `gorm:"type:varchar(255);not null"`
	Description string `gorm:"type:text"`
	Tags        string `gorm:"type:varchar(500)"`
	FolderPath  string `gorm:"type:varchar(500);index:idx_assets_path"`
	ObjectKey   string `gorm:"type:varchar(500);uniqueIndex:idx_assets_object_key"`
	Bucket      string `gorm:"type:varchar(128)"`
	Extension   string `gorm:"type:varchar(32);index:idx_assets_extension"`
	MimeType    string `gorm:"type:varchar(128);index:idx_assets_mime_type"`
	Size        int64
	// Hash 唯一：同一份内容只存一条资产，并发上传同一文件时由数据库裁决去重。
	Hash          string              `gorm:"type:varchar(64);uniqueIndex:idx_assets_hash_unique,where:hash <> ''"`
	Category      asset.AssetCategory `gorm:"type:varchar(32);index:idx_assets_category;default:'OTHER'"`
	Status        asset.AssetStatus   `gorm:"type:varchar(32);index:idx_assets_status;default:'ACTIVE'"`
	Scope         asset.AssetScope    `gorm:"type:varchar(32);not null;index:idx_assets_scope;default:'LIBRARY'"`
	IsPublic      bool                `gorm:"default:false;index:idx_assets_public"`
	DownloadCount int                 `gorm:"default:0"`
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
		FolderPath:    m.FolderPath,
		ObjectKey:     m.ObjectKey,
		Bucket:        m.Bucket,
		Extension:     m.Extension,
		MimeType:      m.MimeType,
		Size:          m.Size,
		Hash:          m.Hash,
		Category:      m.Category,
		Status:        m.Status,
		Scope:         m.Scope,
		IsPublic:      m.IsPublic,
		DownloadCount: m.DownloadCount,
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
		FolderPath:    e.FolderPath,
		ObjectKey:     e.ObjectKey,
		Bucket:        e.Bucket,
		Extension:     e.Extension,
		MimeType:      e.MimeType,
		Size:          e.Size,
		Hash:          e.Hash,
		Category:      e.Category,
		Status:        e.Status,
		Scope:         e.Scope,
		IsPublic:      e.IsPublic,
		DownloadCount: e.DownloadCount,
	}
}

// AssetReferenceModel 业务对象对资产的引用，(asset, owner, field) 唯一。
type AssetReferenceModel struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	AssetID   uint   `gorm:"not null;uniqueIndex:idx_asset_refs_unique,priority:1"`
	OwnerType string `gorm:"type:varchar(64);not null;uniqueIndex:idx_asset_refs_unique,priority:2;index:idx_asset_refs_owner,priority:1"`
	OwnerID   uint   `gorm:"not null;uniqueIndex:idx_asset_refs_unique,priority:3;index:idx_asset_refs_owner,priority:2"`
	Field     string `gorm:"type:varchar(64);not null;uniqueIndex:idx_asset_refs_unique,priority:4"`
	// Name 引用方给文件起的显示名，为空时退回资产自己的文件名
	Name string `gorm:"type:varchar(255);not null;default:''"`
}

func (AssetReferenceModel) TableName() string { return "asset_references" }

func (m AssetReferenceModel) ToEntity() asset.Reference {
	return asset.Reference{OwnerType: m.OwnerType, OwnerID: m.OwnerID, Field: m.Field}
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
	NameEn      string `gorm:"type:varchar(100);not null"`
	NameZh      string `gorm:"type:varchar(100);not null"`
	NameJa      string `gorm:"type:varchar(100);not null"`
	NameKo      string `gorm:"type:varchar(100);not null"`
	Description string `gorm:"type:varchar(500)"`
	IsSystem    bool   `gorm:"default:false"`
	IsPublic    bool   `gorm:"not null;default:false"`
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
		Code: m.Code, Name: shared.Text(m.NameEn, m.NameZh, m.NameJa, m.NameKo),
		Description: m.Description,
		IsSystem:    m.IsSystem, IsPublic: m.IsPublic, IsEnabled: m.IsEnabled, SortOrder: m.SortOrder,
	}
}

type DictItemModel struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	CreatedBy   uint `gorm:"index"`
	UpdatedAt   time.Time
	UpdatedBy   uint   `gorm:"index"`
	TypeCode    string `gorm:"type:varchar(100);index:idx_dict_type_code;not null"`
	LabelEn     string `gorm:"type:varchar(200);not null"`
	LabelZh     string `gorm:"type:varchar(200);not null"`
	LabelJa     string `gorm:"type:varchar(200);not null"`
	LabelKo     string `gorm:"type:varchar(200);not null"`
	Value       string `gorm:"type:varchar(200);not null"`
	Description string `gorm:"type:varchar(500)"`
	Color       string `gorm:"type:varchar(50)"`
	Icon        string `gorm:"type:varchar(100)"`
	IsSystem    bool   `gorm:"not null;default:false"`
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
		TypeCode: m.TypeCode, Label: shared.Text(m.LabelEn, m.LabelZh, m.LabelJa, m.LabelKo),
		Value:       m.Value,
		Description: m.Description, Color: m.Color,
		Icon: m.Icon, IsSystem: m.IsSystem, IsDefault: m.IsDefault,
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
	SendEmail    bool `gorm:"not null;default:false"`
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
		IsGlobal: m.IsGlobal, ExpireAt: m.ExpireAt, SendEmail: m.SendEmail,
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
	DataScope   string `gorm:"size:32;not null;default:'ALL'"`
}

func (RoleModel) TableName() string { return "rbac_roles" }

// RoleDepartmentModel 是 CUSTOM 数据范围指定的部门；部门删除时随之删除。
type RoleDepartmentModel struct {
	RoleID       uint            `gorm:"primaryKey;autoIncrement:false"`
	DepartmentID uint            `gorm:"primaryKey;autoIncrement:false;index"`
	Role         RoleModel       `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Department   DepartmentModel `gorm:"foreignKey:DepartmentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (RoleDepartmentModel) TableName() string { return "rbac_role_departments" }

// DepartmentModel 部门树节点。父子关系用自引用外键 RESTRICT 保证：
// 有下级部门时删不掉；有成员时由 users.department_id 的外键同样拦住。
type DepartmentModel struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	CreatedBy uint `gorm:"index"`
	UpdatedAt time.Time
	UpdatedBy uint             `gorm:"index"`
	ParentID  *uint            `gorm:"index"`
	Parent    *DepartmentModel `gorm:"foreignKey:ParentID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Code      string           `gorm:"uniqueIndex;size:64;not null"`
	Name      string           `gorm:"size:100;not null"`
	SortOrder int              `gorm:"not null;default:0"`
	IsEnabled bool             `gorm:"not null;default:true"`
}

func (DepartmentModel) TableName() string { return "departments" }

func (m *DepartmentModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}
func (m *DepartmentModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *DepartmentModel) ToEntity() department.Department {
	d := department.Department{ParentID: m.ParentID, Code: m.Code, Name: m.Name, SortOrder: m.SortOrder, IsEnabled: m.IsEnabled}
	d.ID, d.CreatedAt, d.CreatedBy, d.UpdatedAt, d.UpdatedBy = m.ID, m.CreatedAt, m.CreatedBy, m.UpdatedAt, m.UpdatedBy
	return d
}

func DepartmentModelFromEntity(e *department.Department) *DepartmentModel {
	return &DepartmentModel{
		ID: e.ID, CreatedAt: e.CreatedAt, CreatedBy: e.CreatedBy, UpdatedAt: e.UpdatedAt, UpdatedBy: e.UpdatedBy,
		ParentID: e.ParentID, Code: e.Code, Name: e.Name, SortOrder: e.SortOrder, IsEnabled: e.IsEnabled,
	}
}

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
		DataScope: permission.DataScope(m.DataScope), DepartmentIDs: []uint{},
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
	ID           string     `gorm:"type:char(36);primaryKey"`
	UserID       uint       `gorm:"not null;index"`
	Username     string     `gorm:"size:100;not null;default:'';index"`
	IpAddr       string     `gorm:"size:64;not null;default:''"`
	UserAgent    string     `gorm:"size:512;not null;default:''"`
	Remember     bool       `gorm:"not null;default:false"`
	CreatedAt    time.Time  `gorm:"not null"`
	LastActiveAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	ExpiresAt    time.Time  `gorm:"not null;index;check:chk_rbac_session_expiry,expires_at > created_at"`
	RevokedAt    *time.Time `gorm:"index"`
	Version      uint64     `gorm:"not null;default:1;check:chk_rbac_session_version,version >= 1"`
	User         UserModel  `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (AuthorizationSessionModel) TableName() string { return "rbac_sessions" }

func (m *AuthorizationSessionModel) ToEntity() permission.AuthorizationSession {
	return permission.AuthorizationSession{
		ID: m.ID, UserID: m.UserID, Username: m.Username, IpAddr: m.IpAddr, UserAgent: m.UserAgent,
		Remember: m.Remember, CreatedAt: m.CreatedAt, LastActiveAt: m.LastActiveAt, ExpiresAt: m.ExpiresAt,
		RevokedAt: m.RevokedAt, Version: m.Version,
	}
}

func AuthorizationSessionModelFromEntity(e *permission.AuthorizationSession) *AuthorizationSessionModel {
	return &AuthorizationSessionModel{
		ID: e.ID, UserID: e.UserID, Username: e.Username, IpAddr: e.IpAddr, UserAgent: e.UserAgent,
		Remember: e.Remember, CreatedAt: e.CreatedAt, LastActiveAt: e.LastActiveAt, ExpiresAt: e.ExpiresAt,
		RevokedAt: e.RevokedAt, Version: e.Version,
	}
}

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
		Code: e.Code, NameEn: e.Name.En, NameZh: e.Name.Zh, NameJa: e.Name.Ja, NameKo: e.Name.Ko,
		Description: e.Description,
		IsSystem:    e.IsSystem, IsPublic: e.IsPublic, IsEnabled: e.IsEnabled, SortOrder: e.SortOrder,
	}
}

func DictItemModelFromEntity(e *dictionary.DictItem) *DictItemModel {
	return &DictItemModel{
		ID: e.ID, CreatedAt: e.CreatedAt, CreatedBy: e.CreatedBy,
		UpdatedAt: e.UpdatedAt, UpdatedBy: e.UpdatedBy,
		TypeCode: e.TypeCode,
		LabelEn:  e.Label.En, LabelZh: e.Label.Zh, LabelJa: e.Label.Ja, LabelKo: e.Label.Ko,
		Value:       e.Value,
		Description: e.Description, Color: e.Color,
		Icon: e.Icon, IsSystem: e.IsSystem, IsDefault: e.IsDefault,
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
		IsGlobal: e.IsGlobal, ExpireAt: e.ExpireAt, SendEmail: e.SendEmail,
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
		IsSystem: e.IsSystem, IsEnabled: e.IsEnabled, DataScope: string(dataScopeOrAll(e.DataScope)),
	}
}

// dataScopeOrAll 让未声明数据范围的角色（旧代码路径、测试夹具）与列默认值一致。
func dataScopeOrAll(scope permission.DataScope) permission.DataScope {
	if scope == "" {
		return permission.DataScopeAll
	}
	return scope
}

func RolePermissionModelFromEntity(e *permission.RolePermission) *RolePermissionModel {
	return &RolePermissionModel{
		ID: e.ID, CreatedAt: e.CreatedAt, CreatedBy: e.CreatedBy,
		RoleID: e.RoleID, ResourceID: e.ResourceID,
		Action: e.Action, IsSystem: e.IsSystem,
	}
}

// ============================================
// Scheduled jobs
// ============================================

// ScheduledJobModel 定时任务的运营设置，主键是代码里注册的任务 key。
type ScheduledJobModel struct {
	Key       string `gorm:"primaryKey;size:64"`
	Cron      string `gorm:"size:100;not null"`
	IsEnabled bool   `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
	UpdatedBy uint
}

func (ScheduledJobModel) TableName() string { return "scheduled_jobs" }

func (m *ScheduledJobModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

func (m *ScheduledJobModel) ToEntity() job.Job {
	return job.Job{Key: m.Key, Cron: m.Cron, IsEnabled: m.IsEnabled, UpdatedAt: m.UpdatedAt, UpdatedBy: m.UpdatedBy}
}

// JobRunModel 一次执行记录；任务 key 不设外键：代码里下线的任务，其历史记录仍可查询。
type JobRunModel struct {
	ID         uint      `gorm:"primarykey"`
	JobKey     string    `gorm:"size:64;not null;index:idx_job_runs_key_started,priority:1"`
	Trigger    string    `gorm:"size:16;not null"`
	Status     string    `gorm:"size:16;not null;index"`
	StartedAt  time.Time `gorm:"not null;index:idx_job_runs_key_started,priority:2"`
	FinishedAt *time.Time
	Affected   int64  `gorm:"not null;default:0"`
	Message    string `gorm:"size:500;not null;default:''"`
	Operator   string `gorm:"size:128;not null;default:''"`
	OperatorID uint   `gorm:"not null;default:0"`
}

func (JobRunModel) TableName() string { return "job_runs" }

func (m *JobRunModel) ToEntity() job.Run {
	return job.Run{
		ID: m.ID, JobKey: m.JobKey, Trigger: job.Trigger(m.Trigger), Status: job.Status(m.Status),
		StartedAt: m.StartedAt, FinishedAt: m.FinishedAt, Affected: m.Affected, Message: m.Message, Operator: m.Operator, OperatorID: m.OperatorID,
	}
}

// ============================================================
// MFA
// ============================================================

// UserTOTPModel 用户的 TOTP 设置；删除用户时一并删除。
type UserTOTPModel struct {
	UserID             uint   `gorm:"primaryKey"`
	SecretCiphertext   string `gorm:"size:255;not null"`
	Enabled            bool   `gorm:"not null;default:false"`
	LastUsedStep       int64  `gorm:"not null;default:0"`
	RecoveryCodeHashes string `gorm:"type:text;not null;default:'[]'"`
	EnabledAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	User               *UserModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (UserTOTPModel) TableName() string { return "user_totp" }

func (m *UserTOTPModel) ToEntity() (*mfa.TOTP, error) {
	var hashes []string
	if err := json.Unmarshal([]byte(m.RecoveryCodeHashes), &hashes); err != nil {
		return nil, err
	}
	return &mfa.TOTP{
		UserID: m.UserID, SecretCiphertext: m.SecretCiphertext, Enabled: m.Enabled, LastUsedStep: m.LastUsedStep,
		RecoveryCodeHashes: hashes, EnabledAt: m.EnabledAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

// ============================================================
// SSO (OIDC)
// ============================================================

// OIDCProviderModel 身份提供方
type OIDCProviderModel struct {
	ID                     uint   `gorm:"primarykey"`
	Code                   string `gorm:"size:32;not null;uniqueIndex"`
	NameEn                 string `gorm:"size:100;not null"`
	NameZh                 string `gorm:"size:100;not null"`
	NameJa                 string `gorm:"size:100;not null"`
	NameKo                 string `gorm:"size:100;not null"`
	Issuer                 string `gorm:"size:500;not null"`
	ClientID               string `gorm:"size:255;not null"`
	ClientSecretCiphertext string `gorm:"size:1024;not null"`
	Scopes                 string `gorm:"size:500;not null"`
	UsernameClaim          string `gorm:"size:64;not null"`
	AutoRegister           bool   `gorm:"not null;default:false"`
	IsEnabled              bool   `gorm:"not null;default:false"`
	SortOrder              int    `gorm:"not null;default:0"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (OIDCProviderModel) TableName() string { return "oidc_providers" }

func (m *OIDCProviderModel) ToEntity() sso.Provider {
	return sso.Provider{
		ID: m.ID, Code: m.Code, Name: shared.I18nText{En: m.NameEn, Zh: m.NameZh, Ja: m.NameJa, Ko: m.NameKo},
		Issuer: m.Issuer, ClientID: m.ClientID, ClientSecretCiphertext: m.ClientSecretCiphertext,
		Scopes: strings.Fields(m.Scopes), UsernameClaim: m.UsernameClaim, AutoRegister: m.AutoRegister,
		IsEnabled: m.IsEnabled, SortOrder: m.SortOrder, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func OIDCProviderModelFromEntity(p *sso.Provider) *OIDCProviderModel {
	return &OIDCProviderModel{
		ID: p.ID, Code: p.Code, NameEn: p.Name.En, NameZh: p.Name.Zh, NameJa: p.Name.Ja, NameKo: p.Name.Ko,
		Issuer: p.Issuer, ClientID: p.ClientID, ClientSecretCiphertext: p.ClientSecretCiphertext,
		Scopes: strings.Join(p.Scopes, " "), UsernameClaim: p.UsernameClaim, AutoRegister: p.AutoRegister,
		IsEnabled: p.IsEnabled, SortOrder: p.SortOrder, CreatedAt: p.CreatedAt,
	}
}

// UserIdentityModel 用户与提供方账号的关联；删除用户或提供方时一并删除。
type UserIdentityModel struct {
	ID          uint   `gorm:"primarykey"`
	UserID      uint   `gorm:"not null;uniqueIndex:idx_user_identities_user_provider,priority:1"`
	ProviderID  uint   `gorm:"not null;uniqueIndex:idx_user_identities_user_provider,priority:2;uniqueIndex:idx_user_identities_provider_subject,priority:1"`
	Subject     string `gorm:"size:255;not null;uniqueIndex:idx_user_identities_provider_subject,priority:2"`
	Email       string `gorm:"size:255;not null;default:''"`
	CreatedAt   time.Time
	LastLoginAt *time.Time
	User        *UserModel         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Provider    *OIDCProviderModel `gorm:"foreignKey:ProviderID;constraint:OnDelete:CASCADE"`
}

func (UserIdentityModel) TableName() string { return "user_identities" }

func (m *UserIdentityModel) ToEntity() sso.Identity {
	return sso.Identity{ID: m.ID, UserID: m.UserID, ProviderID: m.ProviderID, Subject: m.Subject, Email: m.Email, CreatedAt: m.CreatedAt, LastLoginAt: m.LastLoginAt}
}

// NotificationEmailModel 通知邮件发件箱：一条通知对一个收件人一行；删除通知或用户时一并删除。
type NotificationEmailModel struct {
	ID             uint      `gorm:"primarykey"`
	NotificationID uint      `gorm:"not null;uniqueIndex:idx_notification_emails_notification_user,priority:1"`
	UserID         uint      `gorm:"not null;uniqueIndex:idx_notification_emails_notification_user,priority:2"`
	Email          string    `gorm:"size:255;not null"`
	Status         string    `gorm:"size:16;not null;index:idx_notification_emails_due,priority:1"`
	Attempts       int       `gorm:"not null;default:0"`
	LastError      string    `gorm:"size:500;not null;default:''"`
	NextAttemptAt  time.Time `gorm:"not null;index:idx_notification_emails_due,priority:2"`
	SentAt         *time.Time
	CreatedAt      time.Time
	Notification   *NotificationModel `gorm:"foreignKey:NotificationID;constraint:OnDelete:CASCADE"`
	User           *UserModel         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (NotificationEmailModel) TableName() string { return "notification_emails" }

func (m *NotificationEmailModel) ToEntity() notification.EmailDelivery {
	return notification.EmailDelivery{
		ID: m.ID, NotificationID: m.NotificationID, UserID: m.UserID, Email: m.Email,
		Status: notification.EmailStatus(m.Status), Attempts: m.Attempts, LastError: m.LastError,
		NextAttemptAt: m.NextAttemptAt, SentAt: m.SentAt, CreatedAt: m.CreatedAt,
	}
}
