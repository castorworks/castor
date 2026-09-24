package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/hyperits/gosuite/security/hash"
)

// MaxUserImportRows 单个导入文件的数据行上限（与 ErrImportTooManyRows 的文案一致）
const MaxUserImportRows = 1000

// 导入逐行错误的 i18n key
const (
	importUsernameRequired     = "ImportUsernameRequired"
	importUsernameInvalid      = "ImportUsernameInvalid"
	importUsernameTaken        = "ImportUsernameTaken"
	importUsernameDuplicate    = "ImportUsernameDuplicate"
	importNameTooLong          = "ImportNameTooLong"
	importEmailInvalid         = "ImportEmailInvalid"
	importEmailTaken           = "ImportEmailTaken"
	importEmailDuplicate       = "ImportEmailDuplicate"
	importMobileInvalid        = "ImportMobileInvalid"
	importMobileTaken          = "ImportMobileTaken"
	importMobileDuplicate      = "ImportMobileDuplicate"
	importDepartmentNotFound   = "ImportDepartmentNotFound"
	importDepartmentRequired   = "ImportDepartmentRequired"
	importDepartmentOutOfScope = "ImportDepartmentOutOfScope"
	importCreateFailed         = "ImportCreateFailed"
)

// UserImportErrorCodes 全部逐行错误 key，供测试核对翻译是否齐全
var UserImportErrorCodes = []string{
	importUsernameRequired, importUsernameInvalid, importUsernameTaken, importUsernameDuplicate,
	importNameTooLong, importEmailInvalid, importEmailTaken, importEmailDuplicate,
	importMobileInvalid, importMobileTaken, importMobileDuplicate,
	importDepartmentNotFound, importDepartmentRequired, importDepartmentOutOfScope, importCreateFailed,
}

func (svc *userService) Import(ctx context.Context, scope permission.AccessScope, req *dto.UserImportReq) (*dto.UserImportResult, error) {
	if len(req.Rows) == 0 {
		return nil, apperror.ErrImportEmpty
	}
	if len(req.Rows) > MaxUserImportRows {
		return nil, apperror.ErrImportTooManyRows
	}
	plainPassword, err := svc.rsaService.Decrypt(ctx, req.Password)
	if err != nil {
		return nil, ErrPasswordDecryptFailed
	}
	if err := svc.settingHelper.PasswordPolicy(ctx).Validate(plainPassword); err != nil {
		return nil, err
	}

	users, errs, err := svc.validateImportRows(ctx, scope, req.Rows)
	if err != nil {
		return nil, err
	}
	result := &dto.UserImportResult{Errors: errs}
	if len(errs) > 0 {
		return result, apperror.ErrImportInvalidRows
	}

	// 同一批用户共用初始密码，只算一次哈希（bcrypt 很慢，逐个计算 1000 行要一分多钟）。
	// 凭证创建即过期：每个人首次登录都必须换成自己的密码。
	hashed, err := hash.BcryptHashPassword(plainPassword)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i, item := range users {
		item.Password = hashed
		item.CredentialExpireDate = now
		if err := svc.createInternal(ctx, item); err != nil {
			// 校验之后才出现的冲突（并发创建了同名用户等）：之前的行已建好，如实报告。
			result.Errors = append(result.Errors, dto.UserImportError{Line: req.Rows[i].Line, Field: dto.UserImportFieldUsername, Code: importCreateFailed})
			return result, apperror.ErrImportInvalidRows
		}
		result.Created++
		result.CreatedIDs = append(result.CreatedIDs, item.ID)
	}
	return result, nil
}

// validateImportRows 校验每一行并构造待创建的用户；逐行错误收集后一并返回，基础设施错误直接返回。
func (svc *userService) validateImportRows(ctx context.Context, scope permission.AccessScope, rows []dto.UserImportRow) ([]*user.User, []dto.UserImportError, error) {
	departments, err := svc.departments.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	deptByCode := make(map[string]uint, len(departments))
	for _, d := range departments {
		deptByCode[d.Code] = d.ID
	}

	var errs []dto.UserImportError
	fail := func(line int, field, code string) {
		errs = append(errs, dto.UserImportError{Line: line, Field: field, Code: code})
	}
	seen := map[string]map[string]bool{
		dto.UserImportFieldUsername: {}, dto.UserImportFieldEmail: {}, dto.UserImportFieldMobile: {},
	}
	// firstUse 记录文件内第一次出现；重复出现的行报错（大小写不敏感，与唯一索引一致）
	firstUse := func(field, value string) bool {
		key := strings.ToLower(value)
		if seen[field][key] {
			return false
		}
		seen[field][key] = true
		return true
	}

	users := make([]*user.User, 0, len(rows))
	for _, row := range rows {
		item := &user.User{
			Username:          strings.TrimSpace(row.Username),
			Name:              strings.TrimSpace(row.Name),
			AccountSource:     constant.ACCOUNT_SOURCE_INTERNAL,
			Enable:            true,
			AccountExpireDate: time.Now().Add(100 * 365 * 24 * time.Hour),
		}

		switch n := utf8.RuneCountInString(item.Username); {
		case n == 0:
			fail(row.Line, dto.UserImportFieldUsername, importUsernameRequired)
		case n < 3 || n > 100 || item.Username == constant.CASTOR_RESERVED_USER_ROLE_NAME:
			fail(row.Line, dto.UserImportFieldUsername, importUsernameInvalid)
		case !firstUse(dto.UserImportFieldUsername, item.Username):
			fail(row.Line, dto.UserImportFieldUsername, importUsernameDuplicate)
		default:
			taken, err := svc.exists(svc.userRepo.GetByUsername(ctx, item.Username))
			if err != nil {
				return nil, nil, err
			}
			if taken {
				fail(row.Line, dto.UserImportFieldUsername, importUsernameTaken)
			}
		}

		if utf8.RuneCountInString(item.Name) > 100 {
			fail(row.Line, dto.UserImportFieldName, importNameTooLong)
		}

		for _, contact := range []struct {
			field, kind, value, invalid, duplicate, taken string
			lookup                                        func(context.Context, string) (*user.User, error)
		}{
			{dto.UserImportFieldEmail, ContactTypeEmail, strings.TrimSpace(row.Email), importEmailInvalid, importEmailDuplicate, importEmailTaken, svc.userRepo.GetByEmail},
			{dto.UserImportFieldMobile, ContactTypeMobile, strings.TrimSpace(row.Mobile), importMobileInvalid, importMobileDuplicate, importMobileTaken, svc.userRepo.GetByMobile},
		} {
			if contact.value == "" {
				continue
			}
			if validateContact(contact.kind, contact.value) != nil {
				fail(row.Line, contact.field, contact.invalid)
				continue
			}
			if !firstUse(contact.field, contact.value) {
				fail(row.Line, contact.field, contact.duplicate)
				continue
			}
			taken, err := svc.exists(contact.lookup(ctx, contact.value))
			if err != nil {
				return nil, nil, err
			}
			if taken {
				fail(row.Line, contact.field, contact.taken)
				continue
			}
			// 与管理员手工创建一致：管理员填写的联系方式视为已核实。
			if contact.kind == ContactTypeEmail {
				item.Email, item.EmailVerified = contact.value, true
			} else {
				item.Mobile, item.MobileVerified = contact.value, true
			}
		}

		code := strings.TrimSpace(row.DepartmentCode)
		if code == "" {
			// 与 resolveDepartment 相同：数据范围不是"全部"时只能把人建在自己范围内的部门里。
			if !scope.All {
				fail(row.Line, dto.UserImportFieldDepartmentCode, importDepartmentRequired)
			}
		} else if id, ok := deptByCode[code]; !ok {
			fail(row.Line, dto.UserImportFieldDepartmentCode, importDepartmentNotFound)
		} else if !scope.ContainsDepartment(id) {
			fail(row.Line, dto.UserImportFieldDepartmentCode, importDepartmentOutOfScope)
		} else {
			item.DepartmentID = &id
		}

		users = append(users, item)
	}
	return users, errs, nil
}

// exists 把"按唯一键查用户"的结果转成是否已被占用
func (svc *userService) exists(_ *user.User, err error) (bool, error) {
	if errors.Is(err, shared.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}
