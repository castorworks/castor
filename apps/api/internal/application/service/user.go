package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/hyperits/gosuite/logger"
	"github.com/hyperits/gosuite/security/hash"
)

// UserService 用户应用服务接口
//
// 管理端方法都带调用者的数据范围 scope：范围外的用户查不到、改不了（一律按不存在处理），
// 新建或调动用户时目标部门也必须在范围内。
type UserService interface {
	Gets(ctx context.Context, scope permission.AccessScope, page, size int, order string, opts ...query.Option) ([]dto.UserResp, int64, error)
	Get(ctx context.Context, scope permission.AccessScope, id uint) (*dto.UserResp, error)
	GetByUsername(ctx context.Context, username string) (*dto.UserResp, error)
	GetRawByUsername(ctx context.Context, username string) (*user.User, error)
	Post(ctx context.Context, scope permission.AccessScope, req *dto.UserPostReq) (*dto.UserResp, error)
	Put(ctx context.Context, scope permission.AccessScope, id uint, req *dto.UserPutReq) (*dto.UserResp, error)
	Delete(ctx context.Context, scope permission.AccessScope, id uint) error
	// CreateFederated 创建外部身份登录自动注册的账号（没有密码），并授予默认角色
	CreateFederated(ctx context.Context, item *user.User) error
	// Import 批量创建用户：先逐行校验，任一行有误则一个都不建，返回 ErrImportInvalidRows 与逐行错误。
	Import(ctx context.Context, scope permission.AccessScope, req *dto.UserImportReq) (*dto.UserImportResult, error)
}

type userService struct {
	userRepo      user.Repository
	settingHelper *SettingHelper // 共享的系统配置读取辅助工具
	rsaService    RsaService     // RSA 加解密服务
	rbac          RBACService
	assets        AssetReferencer // 头像引用的登记与释放
	departments   department.Repository
}

// NewUserService 创建用户应用服务
func NewUserService(userRepo user.Repository, settingHelper *SettingHelper, rsaService RsaService, rbac RBACService, assets AssetService, departments department.Repository) UserService {
	svc := &userService{
		userRepo:      userRepo,
		settingHelper: settingHelper,
		rsaService:    rsaService,
		rbac:          rbac,
		assets:        assets,
		departments:   departments,
	}
	return svc
}

// withUserScope 把数据范围追加为用户列表的查询条件。
func withUserScope(opts []query.Option, scope permission.AccessScope, column string) []query.Option {
	if option := query.UserScope(column, scope.All, scope.UserID, scope.DepartmentIDs); option != nil {
		opts = append(opts, *option)
	}
	return opts
}

// resolveDepartment 校验调用者把用户放进 departmentID（nil 表示不归属部门）是否合法：
// 部门必须存在；数据范围不是"全部"时，部门必须在范围内，且不能把用户移出部门
// （那会让用户从调用者的范围里消失）。
func (svc *userService) resolveDepartment(ctx context.Context, scope permission.AccessScope, departmentID *uint) error {
	if departmentID == nil {
		if !scope.All {
			return apperror.ErrDataScopeExceedsCaller
		}
		return nil
	}
	if !scope.ContainsDepartment(*departmentID) {
		return apperror.ErrDataScopeExceedsCaller
	}
	all, err := svc.departments.List(ctx)
	if err != nil {
		return err
	}
	for _, d := range all {
		if d.ID == *departmentID {
			return nil
		}
	}
	return apperror.ErrDepartmentNotFound
}

// scopedUser 取出范围内的用户；范围外与不存在同样返回 ErrNotFound，不暴露对方是否存在。
func (svc *userService) scopedUser(ctx context.Context, scope permission.AccessScope, id uint) (*user.User, error) {
	item, err := svc.userRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !scope.Contains(item.ID, item.DepartmentID) {
		return nil, shared.ErrNotFound
	}
	return item, nil
}

func (svc *userService) Gets(ctx context.Context, scope permission.AccessScope, page, size int, order string, opts ...query.Option) ([]dto.UserResp, int64, error) {
	items, total, err := svc.userRepo.Gets(ctx, page, size, order, withUserScope(opts, scope, "id")...)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.UserResp, len(items))
	for i, item := range items {
		if err := resp[i].FromEntity(&item); err != nil {
			return nil, 0, err
		}
	}
	return resp, total, nil
}

func (svc *userService) Get(ctx context.Context, scope permission.AccessScope, id uint) (*dto.UserResp, error) {
	item, err := svc.scopedUser(ctx, scope, id)
	if err != nil {
		return nil, err
	}

	resp := &dto.UserResp{}
	if err := resp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

func (svc *userService) GetByUsername(ctx context.Context, username string) (*dto.UserResp, error) {
	item, err := svc.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	resp := &dto.UserResp{}
	if err := resp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

func (svc *userService) GetRawByUsername(ctx context.Context, username string) (*user.User, error) {
	return svc.userRepo.GetByUsername(ctx, username)
}

func (svc *userService) Post(ctx context.Context, scope permission.AccessScope, req *dto.UserPostReq) (*dto.UserResp, error) {
	if err := svc.resolveDepartment(ctx, scope, req.DepartmentID); err != nil {
		return nil, err
	}
	// RSA 解密密码
	plainPassword, err := svc.rsaService.Decrypt(ctx, req.Password)
	if err != nil {
		return nil, ErrPasswordDecryptFailed
	}

	passwordPolicy := svc.settingHelper.PasswordPolicy(ctx)
	if err := passwordPolicy.Validate(plainPassword); err != nil {
		return nil, err
	}

	item := &user.User{
		Username:     req.Username,
		Name:         req.Name,
		DepartmentID: req.DepartmentID,
	}

	// 管理员填写的联系方式视为已核实（管理员本就掌握非 system 用户的密码，
	// 这里不引入新的权限），但仍要保证非空联系方式全局唯一。
	if err := svc.applyContact(ctx, item, ContactTypeEmail, req.Email); err != nil {
		return nil, err
	}
	if err := svc.applyContact(ctx, item, ContactTypeMobile, req.Mobile); err != nil {
		return nil, err
	}

	item.AccountSource = constant.ACCOUNT_SOURCE_INTERNAL

	cryptoPassword, err := hash.BcryptHashPassword(plainPassword)
	if err != nil {
		return nil, err
	}
	item.Password = cryptoPassword

	item.Enable = true
	item.Locked = false
	item.AccountExpireDate = time.Now().Add(100 * 365 * 24 * time.Hour)
	item.CredentialExpireDate = passwordPolicy.CredentialExpiry(time.Now())

	if err := svc.createInternal(ctx, item); err != nil {
		return nil, err
	}

	resp := &dto.UserResp{}
	if err := resp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

func (svc *userService) CreateFederated(ctx context.Context, item *user.User) error {
	item.Enable = true
	item.AccountExpireDate = time.Now().Add(100 * 365 * 24 * time.Hour)
	return svc.createInternal(ctx, item)
}

// createInternal 写入新用户并授予默认角色；授权失败时删掉刚建的用户，不留半成品。
func (svc *userService) createInternal(ctx context.Context, item *user.User) error {
	if err := svc.userRepo.Create(ctx, item); err != nil {
		return err
	}
	if err := svc.rbac.AddUserRole(ctx, item.Username, permission.RoleUser, true); err != nil {
		if delErr := svc.userRepo.Delete(ctx, item.ID); delErr != nil {
			logger.Errorf("Compensating delete of user %d after role assignment failure failed: %v", item.ID, delErr)
		}
		return err
	}
	return nil
}

func (svc *userService) Put(ctx context.Context, scope permission.AccessScope, id uint, req *dto.UserPutReq) (*dto.UserResp, error) {
	item, err := svc.scopedUser(ctx, scope, id)
	if err != nil {
		return nil, err
	}

	// The built-in system account is managed only through its own account
	// endpoints. Reject every admin-side change to it (including password-only
	// updates) so a delegated user manager cannot reset the superadmin's
	// password and take over.
	if item.Username == "system" {
		return nil, ErrCannotModifySystemUser
	}

	requesterID := ucontext.AuditUserIDFromContext(ctx)
	// Reject self-modification of status fields.
	if req.Enable != nil || req.Locked != nil {
		if requesterID != 0 && requesterID == item.ID {
			return nil, ErrCannotModifyOwnStatus
		}
	}
	if err := svc.ensureCanManage(ctx, requesterID, item.ID); err != nil {
		return nil, err
	}

	if req.Name != "" {
		item.Name = req.Name
	}

	if req.DepartmentID != nil {
		var target *uint
		if *req.DepartmentID != 0 {
			target = req.DepartmentID
		}
		if err := svc.resolveDepartment(ctx, scope, target); err != nil {
			return nil, err
		}
		item.DepartmentID = target
	}

	if req.Email != nil {
		if err := svc.applyContact(ctx, item, ContactTypeEmail, *req.Email); err != nil {
			return nil, err
		}
	}

	if req.Mobile != nil {
		if err := svc.applyContact(ctx, item, ContactTypeMobile, *req.Mobile); err != nil {
			return nil, err
		}
	}

	if req.Password != "" {
		// RSA 解密密码
		plainPassword, err := svc.rsaService.Decrypt(ctx, req.Password)
		if err != nil {
			return nil, ErrPasswordDecryptFailed
		}

		passwordPolicy := svc.settingHelper.PasswordPolicy(ctx)
		if err := passwordPolicy.Validate(plainPassword); err != nil {
			return nil, err
		}
		cryptoPassword, err := hash.BcryptHashPassword(plainPassword)
		if err != nil {
			return nil, err
		}
		item.Password = cryptoPassword
		// 新密码按策略重新计时；同一请求显式给出的凭证过期时间在下面覆盖它。
		item.CredentialExpireDate = passwordPolicy.CredentialExpiry(time.Now())
	}

	// Handle status field updates
	if req.Enable != nil {
		item.Enable = *req.Enable
	}

	if req.Locked != nil {
		item.Locked = *req.Locked
	}

	// Handle expiry date updates (nil = not updating, zero = clear expiry)
	if req.AccountExpireDate != nil {
		item.AccountExpireDate = *req.AccountExpireDate
	}

	if req.CredentialExpireDate != nil {
		item.CredentialExpireDate = *req.CredentialExpireDate
	}

	save := func() error { return svc.userRepo.Update(ctx, item) }
	if req.Avatar != "" && req.Avatar != item.Avatar {
		// 头像是资产对象键：换头像即换引用，旧头像随引用解除而回收。
		err = changeUserAvatar(ctx, svc.assets, item, req.Avatar, save)
	} else {
		err = save()
	}
	if err != nil {
		return nil, err
	}
	if req.Password != "" || req.Enable != nil || req.Locked != nil || req.AccountExpireDate != nil || req.CredentialExpireDate != nil {
		if err := svc.rbac.RevokeUserSessions(ctx, item.ID); err != nil {
			return nil, err
		}
	}

	resp := &dto.UserResp{}
	if err := resp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

// ensureCanManage 拒绝管理拥有调用者所没有权限的用户。Put/Delete 只由管理端调用，
// 上下文里取不到调用者时按失败处理，不能因为身份缺失而跳过检查。
func (svc *userService) ensureCanManage(ctx context.Context, requesterID, targetID uint) error {
	if requesterID == 0 {
		return apperror.ErrTargetUserExceedsCaller
	}
	return svc.rbac.EnsureCanManageUser(ctx, requesterID, targetID)
}

// applyContact 把管理员填写的联系方式写入实体：空串表示解绑，非空串校验格式、
// 确认未被其它账号占用后写入并标记为已验证。
func (svc *userService) applyContact(ctx context.Context, item *user.User, contactType, contact string) error {
	contact = strings.TrimSpace(contact)
	if contact == "" {
		switch contactType {
		case ContactTypeEmail:
			item.Email, item.EmailVerified = "", false
		case ContactTypeMobile:
			item.Mobile, item.MobileVerified = "", false
		}
		return nil
	}

	if err := validateContact(contactType, contact); err != nil {
		return err
	}

	var (
		owner *user.User
		err   error
	)
	switch contactType {
	case ContactTypeEmail:
		owner, err = svc.userRepo.GetByEmail(ctx, contact)
	case ContactTypeMobile:
		owner, err = svc.userRepo.GetByMobile(ctx, contact)
	default:
		return ErrInvalidContactType
	}
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return err
	}
	// 新建用户时 item.ID 为 0，任何已有持有者都会被判定为冲突。
	if err == nil && owner.ID != item.ID {
		return ErrContactAlreadyUsed
	}

	switch contactType {
	case ContactTypeEmail:
		item.Email, item.EmailVerified = contact, true
	case ContactTypeMobile:
		item.Mobile, item.MobileVerified = contact, true
	}
	return nil
}

func (svc *userService) Delete(ctx context.Context, scope permission.AccessScope, id uint) error {
	item, err := svc.scopedUser(ctx, scope, id)
	if err != nil {
		return err
	}

	// Reject deletion of system user
	if item.Username == "system" {
		return ErrCannotDeleteSystemUser
	}

	// Reject self-deletion
	requesterID := ucontext.AuditUserIDFromContext(ctx)
	if requesterID != 0 && requesterID == item.ID {
		return ErrCannotDeleteOwnAccount
	}
	if err := svc.ensureCanManage(ctx, requesterID, item.ID); err != nil {
		return err
	}

	if err := svc.rbac.DeleteUserAuthorization(ctx, id); err != nil {
		return fmt.Errorf("delete authorization for user %s: %w", item.Username, err)
	}
	if err := svc.userRepo.Delete(ctx, id); err != nil {
		return err
	}
	// 用户已删除，释放它引用的资产；失败只会留下可手动清理的附件，不回滚删除。
	if err := svc.assets.DetachOwner(ctx, UserAssetOwnerType, id); err != nil {
		log.WarnCtx(ctx).Err(err).Uint("userID", id).Msg("Failed to release assets of deleted user")
	}
	return nil
}

// DefaultAdminPasswordEnv supplies the initial password of the built-in system user.
const DefaultAdminPasswordEnv = "CASTOR_DEFAULT_ADMIN_PASSWORD"

// MinDefaultAdminPasswordLength is the minimum accepted length of DefaultAdminPasswordEnv.
const MinDefaultAdminPasswordLength = 12

// InitializeDefaultUser is called inside the init-db transaction, never by a server constructor.
//
// The system user's initial password comes from CASTOR_DEFAULT_ADMIN_PASSWORD. Only when
// allowGeneratedPassword is true (development mode) may a random password be generated
// instead; it is returned to the caller so the init-db command can show it once on the
// terminal. It is never written to the application log.
func InitializeDefaultUser(ctx context.Context, repo user.Repository, rbac RBACService, allowGeneratedPassword bool) (string, error) {
	svc := &userService{userRepo: repo, rbac: rbac}
	return svc.createDefaultUserIfNotExists(ctx, allowGeneratedPassword)
}

func (svc *userService) createDefaultUserIfNotExists(ctx context.Context, allowGeneratedPassword bool) (string, error) {
	var generatedPassword string
	_, err := svc.userRepo.GetByUsername(ctx, "system")
	if errors.Is(err, shared.ErrNotFound) {
		rawPassword, generated, err := resolveDefaultPassword(allowGeneratedPassword)
		if err != nil {
			return "", err
		}
		password, err := hash.BcryptHashPassword(rawPassword)
		if err != nil {
			return "", err
		}
		item := &user.User{
			Username:             "system",
			Name:                 "system",
			Password:             password,
			AccountSource:        constant.ACCOUNT_SOURCE_INTERNAL,
			Enable:               true,
			AccountExpireDate:    time.Now().Add(100 * 365 * 24 * time.Hour),
			CredentialExpireDate: time.Now().Add(100 * 365 * 24 * time.Hour),
		}
		if err := svc.userRepo.Create(ctx, item); err != nil {
			return "", err
		}
		if generated {
			generatedPassword = rawPassword
		}
	} else if err != nil {
		return "", err
	}
	if err := svc.rbac.AddUserRole(ctx, "system", "admin", true); err != nil {
		return "", err
	}
	if err := svc.rbac.AddUserRole(ctx, "system", "user", true); err != nil {
		return "", err
	}
	return generatedPassword, nil
}

// resolveDefaultPassword returns the configured initial admin password, or a generated
// one when allowed. generated reports whether the password was generated.
func resolveDefaultPassword(allowGenerated bool) (password string, generated bool, err error) {
	trimmed := strings.TrimSpace(os.Getenv(DefaultAdminPasswordEnv))
	if len(trimmed) >= MinDefaultAdminPasswordLength {
		return trimmed, false, nil
	}
	if !allowGenerated {
		return "", false, fmt.Errorf("%w: set %s to at least %d characters", ErrDefaultAdminPasswordRequired, DefaultAdminPasswordEnv, MinDefaultAdminPasswordLength)
	}
	if trimmed != "" {
		logger.Warnf("%s is shorter than %d characters, generating a random password instead", DefaultAdminPasswordEnv, MinDefaultAdminPasswordLength)
	}
	return generateRandomPassword(20), true, nil
}

// generateRandomPassword creates a random password of the given length using
// uppercase letters, lowercase letters, and digits.
func generateRandomPassword(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// Fallback: should never happen with crypto/rand
			result[i] = charset[i%len(charset)]
			continue
		}
		result[i] = charset[idx.Int64()]
	}
	return string(result)
}
