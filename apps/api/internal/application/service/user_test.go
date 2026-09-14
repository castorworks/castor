package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/hyperits/gosuite/security/hash"
)

type stubRBACService struct{ RBACService }

func (*stubRBACService) DeleteUserAuthorization(context.Context, uint) error     { return nil }
func (*stubRBACService) AddUserRole(context.Context, string, string, bool) error { return nil }

func TestUserService_Post_Validation(t *testing.T) {
	tests := []struct {
		name      string
		req       *dto.UserPostReq
		minLength string
		wantErr   bool
	}{
		{
			name: "valid user creation",
			req: &dto.UserPostReq{
				Username: "testuser",
				Name:     "Test User",
				Password: "password123",
			},
			minLength: "6",
			wantErr:   false,
		},
		{
			name: "password too short",
			req: &dto.UserPostReq{
				Username: "testuser",
				Name:     "Test User",
				Password: "abc",
			},
			minLength: "8",
			wantErr:   true,
		},
		{
			name: "empty password",
			req: &dto.UserPostReq{
				Username: "testuser",
				Name:     "Test User",
				Password: "",
			},
			minLength: "6",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := newMockUserRepo()
			settingRepo := newMockSettingRepo()
			settingRepo.settings[setting.KeySecurityPasswordMinLength] = &setting.Setting{
				Key:   setting.KeySecurityPasswordMinLength,
				Value: tt.minLength,
			}

			svc := &userService{
				userRepo:      userRepo,
				settingHelper: NewSettingHelper(settingRepo),
				rsaService:    &mockRsaService{},
				rbac:          &stubRBACService{},
			}

			resp, err := svc.Post(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Post() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp == nil {
					t.Error("Post() returned nil response for valid input")
					return
				}
				if resp.Username != tt.req.Username {
					t.Errorf("Post() username = %v, want %v", resp.Username, tt.req.Username)
				}
			}
		})
	}
}

func TestUserService_CreateDefaultUserIfNotExists_UsesEnvPassword(t *testing.T) {
	// Set env var with a valid password (≥ 8 chars)
	t.Setenv("CASTOR_DEFAULT_ADMIN_PASSWORD", "MySecurePass123")

	userRepo := newMockUserRepo()
	settingRepo := newMockSettingRepo()
	svc := &userService{
		userRepo:      userRepo,
		settingHelper: NewSettingHelper(settingRepo),
		rsaService:    &mockRsaService{},
		rbac:          &stubRBACService{},
	}

	if _, err := svc.createDefaultUserIfNotExists(context.Background(), false); err != nil {
		t.Fatal(err)
	}

	created, err := userRepo.GetByUsername(context.Background(), "system")
	if err != nil {
		t.Fatalf("default system user was not created: %v", err)
	}

	if created.Name != "system" {
		t.Fatalf("default system user name = %q, want %q", created.Name, "system")
	}

	if !hash.BcryptMatchPassword("MySecurePass123", created.Password) {
		t.Fatal("default system user password does not match env var password")
	}
}

func TestUserService_CreateDefaultUserIfNotExists_GeneratesRandomWhenEnvEmpty(t *testing.T) {
	// Ensure env var is not set
	t.Setenv("CASTOR_DEFAULT_ADMIN_PASSWORD", "")

	userRepo := newMockUserRepo()
	settingRepo := newMockSettingRepo()
	svc := &userService{
		userRepo:      userRepo,
		settingHelper: NewSettingHelper(settingRepo),
		rsaService:    &mockRsaService{},
		rbac:          &stubRBACService{},
	}

	generated, err := svc.createDefaultUserIfNotExists(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}

	if len(generated) != 20 {
		t.Fatalf("generated password must be returned to init-db, got %q", generated)
	}

	created, err := userRepo.GetByUsername(context.Background(), "system")
	if err != nil {
		t.Fatalf("default system user was not created: %v", err)
	}

	// Password should be a random 16-char string; we can't predict it,
	// but we can verify the user was created and has a non-empty hashed password
	if created.Password == "" {
		t.Fatal("default system user password should not be empty")
	}
}

func TestUserService_CreateDefaultUserIfNotExists_GeneratesRandomWhenEnvTooShort(t *testing.T) {
	// Set env var with a short password (< 8 chars)
	t.Setenv("CASTOR_DEFAULT_ADMIN_PASSWORD", "short")

	userRepo := newMockUserRepo()
	settingRepo := newMockSettingRepo()
	svc := &userService{
		userRepo:      userRepo,
		settingHelper: NewSettingHelper(settingRepo),
		rsaService:    &mockRsaService{},
		rbac:          &stubRBACService{},
	}

	generated, err := svc.createDefaultUserIfNotExists(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}

	if len(generated) != 20 {
		t.Fatalf("generated password must be returned to init-db, got %q", generated)
	}

	created, err := userRepo.GetByUsername(context.Background(), "system")
	if err != nil {
		t.Fatalf("default system user was not created: %v", err)
	}

	// The short password should be rejected; a random password is used instead
	if hash.BcryptMatchPassword("short", created.Password) {
		t.Fatal("default system user should NOT use the short env var password")
	}

	if created.Password == "" {
		t.Fatal("default system user password should not be empty")
	}
}

func TestUserService_CreateDefaultUserIfNotExists_SkipsIfUserExists(t *testing.T) {
	t.Setenv("CASTOR_DEFAULT_ADMIN_PASSWORD", "SomePassword123")

	userRepo := newMockUserRepo()
	settingRepo := newMockSettingRepo()

	// Pre-create the system user
	existingPassword, _ := hash.BcryptHashPassword("ExistingPass123")
	userRepo.Create(context.Background(), &user.User{
		Username:      "system",
		Name:          "system",
		Password:      existingPassword,
		AccountSource: "INTERNAL",
		Enable:        true,
	})

	svc := &userService{
		userRepo:      userRepo,
		settingHelper: NewSettingHelper(settingRepo),
		rsaService:    &mockRsaService{},
		rbac:          &stubRBACService{},
	}

	if _, err := svc.createDefaultUserIfNotExists(context.Background(), false); err != nil {
		t.Fatal(err)
	}

	created, err := userRepo.GetByUsername(context.Background(), "system")
	if err != nil {
		t.Fatalf("system user should still exist: %v", err)
	}

	// Password should remain unchanged (the original one)
	if !hash.BcryptMatchPassword("ExistingPass123", created.Password) {
		t.Fatal("system user password should not have been changed when user already exists")
	}
}

func TestUserService_Get_UserStatus(t *testing.T) {
	tests := []struct {
		name      string
		setupRepo func(*mockUserRepo)
		userID    uint
		wantErr   bool
		wantUser  string
	}{
		{
			name: "existing user found",
			setupRepo: func(m *mockUserRepo) {
				m.users["activeuser"] = &user.User{
					ID:       1,
					Username: "activeuser",
					Name:     "Active User",
					Enable:   true,
					Locked:   false,
				}
			},
			userID:   1,
			wantErr:  false,
			wantUser: "activeuser",
		},
		{
			name:      "non-existing user returns error",
			setupRepo: func(m *mockUserRepo) {},
			userID:    999,
			wantErr:   true,
		},
		{
			name: "disabled user still returned",
			setupRepo: func(m *mockUserRepo) {
				m.users["disableduser"] = &user.User{
					ID:       2,
					Username: "disableduser",
					Name:     "Disabled User",
					Enable:   false,
					Locked:   false,
				}
			},
			userID:   2,
			wantErr:  false,
			wantUser: "disableduser",
		},
		{
			name: "locked user still returned",
			setupRepo: func(m *mockUserRepo) {
				m.users["lockeduser"] = &user.User{
					ID:       3,
					Username: "lockeduser",
					Name:     "Locked User",
					Enable:   true,
					Locked:   true,
				}
			},
			userID:   3,
			wantErr:  false,
			wantUser: "lockeduser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := newMockUserRepo()
			settingRepo := newMockSettingRepo()

			if tt.setupRepo != nil {
				tt.setupRepo(userRepo)
			}

			svc := &userService{
				userRepo:      userRepo,
				settingHelper: NewSettingHelper(settingRepo),
				rsaService:    &mockRsaService{},
			}

			resp, err := svc.Get(context.Background(), tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp == nil {
					t.Error("Get() returned nil response")
					return
				}
				if resp.Username != tt.wantUser {
					t.Errorf("Get() username = %v, want %v", resp.Username, tt.wantUser)
				}
			}
		})
	}
}

func TestUserService_Delete(t *testing.T) {
	tests := []struct {
		name      string
		setupRepo func(*mockUserRepo)
		userID    uint
		wantErr   bool
	}{
		{
			name: "delete existing user",
			setupRepo: func(m *mockUserRepo) {
				m.users["testuser"] = &user.User{
					ID:       1,
					Username: "testuser",
				}
			},
			userID:  1,
			wantErr: false,
		},
		{
			name:      "delete non-existing user returns error",
			setupRepo: func(m *mockUserRepo) {},
			userID:    999,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := newMockUserRepo()
			settingRepo := newMockSettingRepo()

			if tt.setupRepo != nil {
				tt.setupRepo(userRepo)
			}

			svc := &userService{
				userRepo:      userRepo,
				settingHelper: NewSettingHelper(settingRepo),
				rsaService:    &mockRsaService{},
				rbac:          &stubRBACService{},
			}

			err := svc.Delete(context.Background(), tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify user is actually deleted
				_, err := svc.Get(context.Background(), tt.userID)
				if !errors.Is(err, shared.ErrNotFound) {
					t.Error("Delete() user still exists after deletion")
				}
			}
		})
	}
}

func TestUserService_GetByUsername(t *testing.T) {
	tests := []struct {
		name      string
		setupRepo func(*mockUserRepo)
		username  string
		wantErr   bool
	}{
		{
			name: "existing username",
			setupRepo: func(m *mockUserRepo) {
				m.users["testuser"] = &user.User{
					ID:       1,
					Username: "testuser",
					Name:     "Test User",
				}
			},
			username: "testuser",
			wantErr:  false,
		},
		{
			name:      "non-existing username",
			setupRepo: func(m *mockUserRepo) {},
			username:  "nonexistent",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := newMockUserRepo()
			settingRepo := newMockSettingRepo()

			if tt.setupRepo != nil {
				tt.setupRepo(userRepo)
			}

			svc := &userService{
				userRepo:      userRepo,
				settingHelper: NewSettingHelper(settingRepo),
				rsaService:    &mockRsaService{},
			}

			resp, err := svc.GetByUsername(context.Background(), tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resp.Username != tt.username {
				t.Errorf("GetByUsername() username = %v, want %v", resp.Username, tt.username)
			}
		})
	}
}

type bootstrapFailureRepo struct {
	user.Repository
	lookupErr error
	createErr error
}

func (r *bootstrapFailureRepo) GetByUsername(context.Context, string) (*user.User, error) {
	return nil, r.lookupErr
}
func (r *bootstrapFailureRepo) Create(context.Context, *user.User) error { return r.createErr }

func TestInitializeDefaultUserPropagatesFailure(t *testing.T) {
	t.Setenv(DefaultAdminPasswordEnv, "ValidPassword123")
	expected := errors.New("database unavailable")
	for _, tc := range []struct {
		name string
		repo *bootstrapFailureRepo
	}{
		{"lookup", &bootstrapFailureRepo{lookupErr: expected}},
		{"create", &bootstrapFailureRepo{lookupErr: shared.ErrNotFound, createErr: expected}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := InitializeDefaultUser(context.Background(), tc.repo, &stubRBACService{}, false)
			if !errors.Is(err, expected) {
				t.Fatalf("got %v, want %v", err, expected)
			}
		})
	}
}

func TestNewUserServiceDoesNotInitializeDatabase(t *testing.T) {
	repo := newMockUserRepo()
	NewUserService(repo, nil, nil, &stubRBACService{})
	if _, err := repo.GetByUsername(context.Background(), "system"); !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("constructor must not create system user: %v", err)
	}
}

func TestUserService_CreateDefaultUserRequiresPasswordOutsideDevelopment(t *testing.T) {
	for _, value := range []string{"", "short-pass"} {
		t.Setenv(DefaultAdminPasswordEnv, value)
		userRepo := newMockUserRepo()
		svc := &userService{userRepo: userRepo, rbac: &stubRBACService{}}
		generated, err := svc.createDefaultUserIfNotExists(context.Background(), false)
		if !errors.Is(err, ErrDefaultAdminPasswordRequired) {
			t.Fatalf("env %q: error = %v, want ErrDefaultAdminPasswordRequired", value, err)
		}
		if generated != "" || len(userRepo.created) != 0 {
			t.Fatalf("env %q: no user or password may be created", value)
		}
	}
}
