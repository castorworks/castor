package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
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

var migrations = []migration{
	{
		Version: 1,
		Name:    "initial_schema",
		Up:      AutoMigrate,
	},
}

// Migrate applies each registered schema version once. The first version is
// intentionally the existing GORM schema so current installations can adopt
// version tracking without data loss. New schema changes must be added as a
// new migration entry and must not be hidden behind a recurring AutoMigrate.
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

	// Databases created before schema version tracking already contain the
	// managed tables. Reconcile them once, then record the baseline version.
	if len(applied) == 0 && db.Migrator().HasTable("users") {
		if err := AutoMigrate(db); err != nil {
			return err
		}
		if err := db.Create(&SchemaMigration{Version: 1, Name: "legacy_baseline", AppliedAt: time.Now().UTC()}).Error; err != nil {
			return err
		}
		appliedVersions[1] = true
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

// AutoMigrate 执行所有表结构的自动迁移（使用 persistence models）
func AutoMigrate(db *gorm.DB) error {
	logger.Infof("Running database auto-migration...")

	if err := db.AutoMigrate(
		&models.UserModel{},
		&models.AssetModel{},
		&models.LoginHistoryModel{},
		&models.AuditLogModel{},
		&models.ResourceModel{},
		&models.MenuModel{},
		&models.MenuPermissionModel{},
		&models.RoleModel{},
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
	); err != nil {
		return err
	}

	logger.Infof("Database auto-migration completed")

	if err := seedSettings(db); err != nil {
		return err
	}
	if err := seedDictTypes(db); err != nil {
		return err
	}
	return seedDictItems(db)
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

func seedDictTypes(db *gorm.DB) error {
	for _, dt := range dictionary.DefaultDictTypes {
		var existing models.DictTypeModel
		if err := db.Where("code = ?", dt.Code).First(&existing).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			m := &models.DictTypeModel{
				Code: dt.Code, Name: dt.Name, Description: dt.Description,
				IsSystem: dt.IsSystem, IsEnabled: dt.IsEnabled, SortOrder: dt.SortOrder,
			}
			if err := db.Create(m).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func seedDictItems(db *gorm.DB) error {
	for _, di := range dictionary.DefaultDictItems {
		var existing models.DictItemModel
		if err := db.Where("type_code = ? AND value = ?", di.TypeCode, di.Value).First(&existing).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			m := &models.DictItemModel{
				TypeCode: di.TypeCode, Label: di.Label, Value: di.Value,
				Description: di.Description, Extra: di.Extra, Color: di.Color,
				Icon: di.Icon, ParentID: di.ParentID, IsDefault: di.IsDefault,
				IsEnabled: di.IsEnabled, SortOrder: di.SortOrder,
			}
			if err := db.Create(m).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}
