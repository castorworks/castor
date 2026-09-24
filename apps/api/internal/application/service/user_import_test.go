package service

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/permission"
)

const importPassword = "Import-Passw0rd!"

func TestUserService_ImportCreatesUsersThatMustChangePassword(t *testing.T) {
	svc, repo := newUserScopeFixture()
	before := time.Now()
	result, err := svc.Import(managerCtx(), fullScope, &dto.UserImportReq{Password: importPassword, Rows: []dto.UserImportRow{
		{Line: 2, Username: " amy ", Name: "Amy", Email: "amy@example.com", DepartmentCode: "rd"},
		{Line: 3, Username: "ben", Mobile: "13800000001"},
	}})
	if err != nil {
		t.Fatalf("Import() err = %v (%+v)", err, result)
	}
	if result.Created != 2 || len(result.CreatedIDs) != 2 || len(result.Errors) != 0 {
		t.Fatalf("result = %+v", result)
	}
	amy, ben := repo.users["amy"], repo.users["ben"]
	if amy == nil || ben == nil {
		t.Fatalf("users not created: %v", repo.users)
	}
	if amy.DepartmentID == nil || *amy.DepartmentID != 2 || !amy.EmailVerified || amy.Name != "Amy" || !ben.MobileVerified {
		t.Fatalf("amy = %+v, ben = %+v", amy, ben)
	}
	for _, u := range []string{"amy", "ben"} {
		item := repo.users[u]
		if !item.Enable || item.Password == "" || item.Password == importPassword {
			t.Fatalf("%s: password must be stored hashed and the account enabled: %+v", u, item)
		}
		// 共用的初始密码必须在首次登录时换掉
		if item.CredentialExpireDate.After(time.Now()) || item.CredentialExpireDate.Before(before) {
			t.Fatalf("%s: credential should expire at import time, got %v", u, item.CredentialExpireDate)
		}
	}
}

func TestUserService_ImportRejectsTheWholeFileOnAnyInvalidRow(t *testing.T) {
	svc, repo := newUserScopeFixture()
	repo.users["inside"].Email = "taken@example.com"
	existing := len(repo.users)
	rows := []dto.UserImportRow{
		{Line: 2, Username: "valid-one", DepartmentCode: "rd"},
		{Line: 3, Username: ""},
		{Line: 4, Username: "ab"},
		{Line: 5, Username: "inside"},
		{Line: 6, Username: "Twin", Email: "not-an-email"},
		{Line: 7, Username: "twin", Email: "taken@example.com", Mobile: "123"},
		{Line: 8, Username: "dup-contact-a", Email: "same@example.com"},
		{Line: 9, Username: "dup-contact-b", Email: "SAME@example.com", DepartmentCode: "nowhere"},
		{Line: 10, Username: "castor-internal", Name: string(make([]rune, 101))},
	}
	result, err := svc.Import(managerCtx(), fullScope, &dto.UserImportReq{Password: importPassword, Rows: rows})
	if !errors.Is(err, apperror.ErrImportInvalidRows) {
		t.Fatalf("err = %v; want ErrImportInvalidRows", err)
	}
	if result.Created != 0 || len(repo.users) != existing {
		t.Fatalf("nothing may be created when a row is invalid; created %d", result.Created)
	}
	got := map[[2]any]string{}
	for _, e := range result.Errors {
		got[[2]any{e.Line, e.Field}] = e.Code
	}
	want := map[[2]any]string{
		{3, "username"}:       importUsernameRequired,
		{4, "username"}:       importUsernameInvalid,
		{5, "username"}:       importUsernameTaken,
		{6, "email"}:          importEmailInvalid,
		{7, "username"}:       importUsernameDuplicate,
		{7, "email"}:          importEmailTaken,
		{7, "mobile"}:         importMobileInvalid,
		{9, "email"}:          importEmailDuplicate,
		{9, "departmentCode"}: importDepartmentNotFound,
		{10, "username"}:      importUsernameInvalid,
		{10, "name"}:          importNameTooLong,
	}
	for key, code := range want {
		if got[key] != code {
			t.Errorf("line %v field %v: code = %q, want %q", key[0], key[1], got[key], code)
		}
	}
	if len(result.Errors) != len(want) {
		t.Errorf("errors = %+v; want exactly %d", result.Errors, len(want))
	}
}

func TestUserService_ImportRespectsDataScope(t *testing.T) {
	svc, _ := newUserScopeFixture()
	scoped := permission.AccessScope{UserID: managerID, DepartmentIDs: []uint{2}}
	result, err := svc.Import(managerCtx(), scoped, &dto.UserImportReq{Password: importPassword, Rows: []dto.UserImportRow{
		{Line: 2, Username: "no-dept"},
		{Line: 3, Username: "other-dept", DepartmentCode: "sales"},
		{Line: 4, Username: "own-dept", DepartmentCode: "rd"},
	}})
	if !errors.Is(err, apperror.ErrImportInvalidRows) {
		t.Fatalf("err = %v", err)
	}
	if len(result.Errors) != 2 || result.Errors[0].Code != importDepartmentRequired || result.Errors[1].Code != importDepartmentOutOfScope {
		t.Fatalf("errors = %+v", result.Errors)
	}
}

func TestUserService_ImportRejectsBadRequests(t *testing.T) {
	svc, _ := newUserScopeFixture()
	ctx := managerCtx()
	if _, err := svc.Import(ctx, fullScope, &dto.UserImportReq{Password: importPassword}); !errors.Is(err, apperror.ErrImportEmpty) {
		t.Errorf("empty err = %v", err)
	}
	tooMany := make([]dto.UserImportRow, MaxUserImportRows+1)
	if _, err := svc.Import(ctx, fullScope, &dto.UserImportReq{Password: importPassword, Rows: tooMany}); !errors.Is(err, apperror.ErrImportTooManyRows) {
		t.Errorf("too many rows err = %v", err)
	}
	if _, err := svc.Import(ctx, fullScope, &dto.UserImportReq{Password: "x", Rows: []dto.UserImportRow{{Line: 2, Username: "weak"}}}); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("weak password err = %v; the policy must apply to the shared initial password", err)
	}
}

// 每个逐行错误 key 都要有四语翻译，否则界面上只能显示 key。
func TestUserImportErrorCodesAreTranslated(t *testing.T) {
	keyPattern := regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_]*)\s*=`)
	for _, locale := range []string{"zh", "en", "ja", "ko"} {
		f, err := os.Open(filepath.Join("..", "..", "..", "configs", "i18n", locale+".toml"))
		if err != nil {
			t.Fatal(err)
		}
		keys := map[string]bool{}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if m := keyPattern.FindStringSubmatch(scanner.Text()); m != nil {
				keys[m[1]] = true
			}
		}
		f.Close()
		for _, code := range UserImportErrorCodes {
			if !keys[code] {
				t.Errorf("%s.toml is missing %s", locale, code)
			}
		}
	}
}
