package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/hyperits/gosuite/security/hash"
)

// H1: an admin-side update must not be able to reset the built-in system
// superadmin's password (or otherwise modify it).
func TestUserService_Put_RejectsSystemUserModification(t *testing.T) {
	userRepo := newMockUserRepo()
	original, _ := hash.BcryptHashPassword("OriginalPass123")
	userRepo.users["system"] = &user.User{ID: 1, Username: "system", Name: "system", Password: original, Enable: true}

	svc := &userService{
		userRepo:      userRepo,
		settingHelper: NewSettingHelper(newMockSettingRepo()),
		rsaService:    &mockRsaService{},
		rbac:          &stubRBACService{},
	}

	_, err := svc.Put(context.Background(), fullScope, 1, &dto.UserPutReq{Password: "NewAttackerPass123"})
	if !errors.Is(err, ErrCannotModifySystemUser) {
		t.Fatalf("expected ErrCannotModifySystemUser, got %v", err)
	}
	if userRepo.users["system"].Password != original {
		t.Fatal("system password must be unchanged after a rejected update")
	}
}

// M1: system resources must not have their Path/Actions repointed.
func TestPermissionService_UpdateResource_FreezesSystemPathAndActions(t *testing.T) {
	existing := permission.Resource{ID: 1, Code: "admin:users:update", Path: "/api/v1/admin/users/:id", Actions: permission.StringSlice{"PUT"}, IsSystem: true, IsEnabled: true}
	repo := &fixResourceRepo{resource: existing}
	svc := &permissionService{resourceRepo: repo, authzRepo: &fixAuthzRepo{}}

	// Repointing the path of a system resource is rejected.
	err := svc.UpdateResource(context.Background(), &permission.Resource{ID: 1, Code: "admin:users:update", Name: "Update Users", Category: permission.CategoryAdmin, Path: "/api/v1/unused", Actions: permission.StringSlice{"PUT"}})
	if !errors.Is(err, ErrCannotModifySystemResource) {
		t.Fatalf("expected ErrCannotModifySystemResource for path change, got %v", err)
	}
	// Changing the actions of a system resource is rejected.
	err = svc.UpdateResource(context.Background(), &permission.Resource{ID: 1, Code: "admin:users:update", Name: "Update Users", Category: permission.CategoryAdmin, Path: "/api/v1/admin/users/:id", Actions: permission.StringSlice{"GET"}})
	if !errors.Is(err, ErrCannotModifySystemResource) {
		t.Fatalf("expected ErrCannotModifySystemResource for action change, got %v", err)
	}
}

type fixResourceRepo struct {
	permission.ResourceRepository
	resource permission.Resource
}

func (r *fixResourceRepo) Get(context.Context, uint) (*permission.Resource, error) {
	cp := r.resource
	return &cp, nil
}

// fixAuthzRepo runs the WithTx closure inline with itself as the tx handle.
type fixAuthzRepo struct {
	permission.AuthorizationRepository
}

func (r *fixAuthzRepo) WithTx(ctx context.Context, fn func(permission.AuthorizationRepository) error) error {
	return fn(r)
}
func (r *fixAuthzRepo) LockRoles(context.Context) error { return nil }

// H3: avatar upload must sniff bytes and reject content whose real type is not a
// permitted image, even when the client declares image/png with a .png name.
type fakeMultipartFile struct{ *bytes.Reader }

func (fakeMultipartFile) Close() error { return nil }

func TestUserAvatarService_Upload_RejectsNonImageBytes(t *testing.T) {
	repo := newFakeAssetRepo()
	storage := newFakeAssetStorage()
	users := newMockUserRepo()
	users.users["alice"] = &user.User{ID: 7, Username: "alice"}
	svc := NewUserAvatarService(NewAssetService(storage, repo, AssetPolicy{}, nil), users)

	body := []byte("<html><script>alert(1)</script></html>")
	f := fakeMultipartFile{bytes.NewReader(body)}

	_, err := svc.Upload(context.Background(), 7, "avatar.png", f, "image/png", int64(len(body)))
	if !errors.Is(err, apperror.ErrAssetInvalidType) {
		t.Fatalf("expected ErrAssetInvalidType for HTML disguised as PNG, got %v", err)
	}
	if len(storage.objects) != 0 || len(repo.assets) != 0 {
		t.Fatal("rejected content must never reach storage or the asset table")
	}
}

// codeType 大小写不敏感：绑定层与服务端常量的大小写一旦不一致，/auth/code 就会完全不可用。
// 归一化后两种形式都必须进入正确分支。
func TestAuthService_PostCode_CodeTypeIsCaseInsensitive(t *testing.T) {
	newSvc := func() *authService {
		return &authService{
			rateLimiter: &mockRateLimiter{},
			captcha:     &mockCaptchaClient{valid: true},
			verifyCode:  &mockVerifyCodeClient{valid: true},
			policy:      RuntimePolicy{Development: false},
		}
	}

	// 进入 EMAIL 分支的证明：邮箱格式校验被触发（而不是 ErrInvalidCodeType）。
	for _, codeType := range []string{"EMAIL", "email", " Email "} {
		t.Run("email/"+codeType, func(t *testing.T) {
			err := newSvc().PostCode(context.Background(),
				&dto.ConfirmCodeReq{CodeType: codeType, Username: "not-an-email"}, "203.0.113.5")
			if !errors.Is(err, ErrInvalidEmailFormat) {
				t.Fatalf("codeType %q should reach the EMAIL branch, got %v", codeType, err)
			}
		})
	}

	// 进入 MOBILE 分支的证明：手机号格式校验被触发。
	for _, codeType := range []string{"MOBILE", "mobile"} {
		t.Run("mobile/"+codeType, func(t *testing.T) {
			err := newSvc().PostCode(context.Background(),
				&dto.ConfirmCodeReq{CodeType: codeType, Username: "not-a-phone"}, "203.0.113.5")
			if !errors.Is(err, ErrInvalidPhoneFormat) {
				t.Fatalf("codeType %q should reach the MOBILE branch, got %v", codeType, err)
			}
		})
	}

	// 无法识别的类型仍应明确报错。
	err := newSvc().PostCode(context.Background(),
		&dto.ConfirmCodeReq{CodeType: "carrier-pigeon", Username: "alice@example.com"}, "203.0.113.5")
	if !errors.Is(err, ErrInvalidCodeType) {
		t.Fatalf("unknown codeType should return ErrInvalidCodeType, got %v", err)
	}
}
