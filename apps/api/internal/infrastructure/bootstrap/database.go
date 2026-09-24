package bootstrap

import (
	"context"

	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/infrastructure/authorization"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/infrastructure/database"
	"github.com/castorworks/castor/internal/infrastructure/persistence"
)

// Result reports facts about an init-db run that the operator must see.
type Result struct {
	// GeneratedAdminPassword is set only when the system user was created in development
	// mode without CASTOR_DEFAULT_ADMIN_PASSWORD. It must be shown once and never logged.
	GeneratedAdminPassword string
}

// Initialize serializes schema and seed changes across concurrent deployments.
// The transaction-scoped lock is released by PostgreSQL on commit or rollback.
func Initialize(ctx context.Context, db *gorm.DB) (Result, error) {
	var result Result
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(0x434153544f52)).Error; err != nil {
			return err
		}
		generated, err := initialize(tx)
		result.GeneratedAdminPassword = generated
		return err
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func initialize(tx *gorm.DB) (string, error) {
	if err := database.Migrate(tx); err != nil {
		return "", err
	}
	// 字典与菜单一样每次对账：新增的默认类型/字典项重跑 init-db 即可进入已有库。
	if err := database.SeedDictionary(tx); err != nil {
		return "", err
	}
	resources := persistence.NewResourceRepository(tx)
	roles := persistence.NewRoleRepository(tx)
	authz := persistence.NewAuthorizationRepository(tx)
	users := persistence.NewUserRepository(tx)
	ctx := tx.Statement.Context
	if err := authorization.InitializeDefaults(ctx, resources, roles, authz); err != nil {
		return "", err
	}
	if err := initializeMenus(ctx, persistence.NewMenuRepository(tx), resources); err != nil {
		return "", err
	}
	rbac := service.NewRBACService(authz, roles, resources, users, persistence.NewDepartmentRepository(tx))
	return service.InitializeDefaultUser(ctx, users, rbac, config.C.General.Development)
}
