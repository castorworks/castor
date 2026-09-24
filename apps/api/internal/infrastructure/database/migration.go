package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"github.com/hyperits/gosuite/logger"
	"gorm.io/gorm"
)

const CurrentSchemaVersion uint = 1

// SchemaMigration records the schema versions applied to a database. The
// advisory lock held by bootstrap.Initialize serializes writes to this table.
type SchemaMigration struct {
	Version   uint      `gorm:"primaryKey"`
	Name      string    `gorm:"type:varchar(128);not null"`
	AppliedAt time.Time `gorm:"not null"`
}

func (SchemaMigration) TableName() string { return "schema_migrations" }

type migration struct {
	Version uint
	Name    string
	Up      func(*gorm.DB) error
}

// migrations 按版本顺序登记结构变更。新的变更追加一个版本并同步 CurrentSchemaVersion，
// 已发布的版本不再修改。
var migrations = []migration{
	{Version: 1, Name: "initial_schema", Up: initialSchema},
}

// usersDepartmentForeignKey 让仍有成员的部门删不掉（ON DELETE RESTRICT）。
const usersDepartmentForeignKey = "fk_users_department"

// userContactUniqueIndexes 用部分唯一索引表达“联系方式可以为空，一旦填写则全局唯一”。
// 普通唯一索引做不到这一点：这里的空值是空串，会互相冲突，必须带 WHERE 谓词把未绑定的行排除在外。
var userContactUniqueIndexes = []struct {
	Name      string
	Statement string
}{
	{"idx_users_email_unique", `CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users (email) WHERE email <> ''`},
	{"idx_users_mobile_unique", `CREATE UNIQUE INDEX IF NOT EXISTS idx_users_mobile_unique ON users (mobile) WHERE mobile <> ''`},
}

// initialSchema（schema version 1）按 persistence models 建出全部表，再补上 GORM 标签表达不了的部分：
//   - 用户邮箱/手机号的部分唯一索引；
//   - users.department_id 与 asset_references.asset_id 的 RESTRICT 外键：有成员的部门、
//     被业务记录引用的资产都删不掉，服务层的先查后删挡不住并发，外键是最终防线；
//   - deployment：每个数据库一个随机、永久的部署标识。多套 Castor 共用 Redis 时，API 启动用它
//     认领自己的 key 前缀（见 cache.ClaimNamespace），误配相同 InstanceID 的另一套部署会被拒绝启动。
//     标识存在库里而不是取自连接地址，换端口、换主机名、整库迁移都不会让部署"变成另一套"。
func initialSchema(db *gorm.DB) error {
	if err := AutoMigrate(db); err != nil {
		return err
	}
	statements := []string{
		userContactUniqueIndexes[0].Statement,
		userContactUniqueIndexes[1].Statement,
		`ALTER TABLE users ADD CONSTRAINT ` + usersDepartmentForeignKey +
			` FOREIGN KEY (department_id) REFERENCES departments(id) ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE asset_references ADD CONSTRAINT fk_asset_references_asset
			FOREIGN KEY (asset_id) REFERENCES assets (id) ON DELETE RESTRICT`,
		`CREATE TABLE deployment (
			singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
			id uuid NOT NULL,
			created_at timestamptz NOT NULL DEFAULT now()
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("initial schema: %w", err)
		}
	}
	if err := db.Exec(`INSERT INTO deployment (id) VALUES (?)`, uuid.NewString()).Error; err != nil {
		return fmt.Errorf("deployment identity: %w", err)
	}
	return nil
}

// DeploymentID 返回本数据库的部署标识；库尚未初始化时提示先执行 init-db。
func DeploymentID(ctx context.Context, db *gorm.DB) (string, error) {
	var id string
	if err := db.WithContext(ctx).Raw(`SELECT id::text FROM deployment`).Scan(&id).Error; err != nil {
		return "", fmt.Errorf("read deployment identity (run init-db first): %w", err)
	}
	if id == "" {
		return "", errors.New("deployment identity is missing: run init-db first")
	}
	return id, nil
}

// Migrate applies each registered schema version once, in order.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&SchemaMigration{}); err != nil {
		return err
	}

	var applied []SchemaMigration
	if err := db.Order("version ASC").Find(&applied).Error; err != nil {
		return err
	}
	appliedVersions := make(map[uint]bool, len(applied))
	for _, record := range applied {
		if record.Version > CurrentSchemaVersion {
			return fmt.Errorf("database schema version %d is newer than application version %d", record.Version, CurrentSchemaVersion)
		}
		appliedVersions[record.Version] = true
	}

	for _, item := range migrations {
		if appliedVersions[item.Version] {
			continue
		}
		if err := item.Up(db); err != nil {
			return err
		}
		if err := db.Create(&SchemaMigration{Version: item.Version, Name: item.Name, AppliedAt: time.Now().UTC()}).Error; err != nil {
			return err
		}
	}
	return nil
}

// AutoMigrate 按 persistence models 建表，并写入默认设置与字典。
func AutoMigrate(db *gorm.DB) error {
	logger.Infof("Running database auto-migration...")

	if err := db.AutoMigrate(
		&models.UserModel{},
		&models.AssetModel{},
		&models.AssetReferenceModel{},
		&models.LoginHistoryModel{},
		&models.AuditLogModel{},
		&models.ResourceModel{},
		&models.MenuModel{},
		&models.MenuPermissionModel{},
		&models.RoleModel{},
		&models.DepartmentModel{},
		&models.RoleDepartmentModel{},
		&models.RolePermissionModel{},
		&models.UserRoleModel{},
		&models.RoleHierarchyModel{},
		&models.SeparationConstraintModel{},
		&models.ConstraintRoleModel{},
		&models.AuthorizationSessionModel{},
		&models.SessionRoleModel{},
		&models.SettingModel{},
		&models.DictTypeModel{},
		&models.DictItemModel{},
		&models.NotificationModel{},
		&models.UserNotificationModel{},
		&models.ScheduledJobModel{},
		&models.JobRunModel{},
		&models.UserTOTPModel{},
		&models.OIDCProviderModel{},
		&models.UserIdentityModel{},
		&models.NotificationEmailModel{},
	); err != nil {
		return err
	}

	logger.Infof("Database auto-migration completed")

	if err := seedSettings(db); err != nil {
		return err
	}
	return seedDictionaryRows(db)
}

func seedSettings(db *gorm.DB) error {
	for _, s := range setting.DefaultSettings {
		var existing models.SettingModel
		if err := db.Where("key = ?", s.Key).First(&existing).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			m := &models.SettingModel{
				Key: s.Key, Value: s.Value, Type: s.Type, Category: s.Category,
				Name: s.Name, Description: s.Description, DefaultVal: s.DefaultVal,
				Options: s.Options, IsSystem: s.IsSystem, IsPublic: s.IsPublic,
				SortOrder: s.SortOrder,
			}
			if err := db.Create(m).Error; err != nil {
				return err
			} else {
				logger.Infof("Created default setting: %s", s.Key)
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}
