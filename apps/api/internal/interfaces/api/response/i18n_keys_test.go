package response

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

var requiredI18nKeys = []string{
	InfMenuSaved, InfMenuDeleted,
	ErrInvalidMenu,
	ErrMenuNotFound,
	ErrMenuConflict,
	ErrMenuHierarchy,
	ErrMenuHasChildren,
	ErrMenuPermission,
	InfCreateSuccess,
	InfUpdateSuccess,
	InfDeleteSuccess,
	InfBatchDeleteSuccess,
	InfUploadSuccess,
	InfMoveSuccess,
	InfStatusUpdated,
	InfBatchStatusUpdated,
	InfUserCreated,
	InfPasswordChanged,
	InfPasswordReset,
	InfCodeSent,
	InfFileUploaded,
	InfFileUpdated,
	InfFileDeleted,
	InfFileDuplicate,
	InfRoleCreated,
	InfRoleUpdated,
	InfRoleDeleted,
	InfPermissionUpdated,
	InfDictCreated,
	InfDictUpdated,
	InfDictDeleted,
	InfSettingUpdated,
	InfNotificationSent,
	InfNotificationRead,
	InfAllMarkedRead,
	ErrNotificationRecipientsRequired,
	ErrNotificationNotDelivered,
	ErrUserNotFound,
	ErrEmailAlreadyExists,
	ErrCurrentPasswordIncorrect,
	ErrFileUploadFailed,
	ErrFileTooLarge,
	ErrRequestTooLarge,
	ErrInvalidFilename,
	ErrInvalidFileType,
	ErrAssetNotFound,
	ErrInvalidID,
	ErrObjectKeyRequired,
	ErrRoleNotFound,
	ErrRoleCodeExists,
	ErrPermissionDenied,
	ErrResourceNotFound,
	ErrResourceCodeExists,
	ErrInvalidResource,
	ErrResourcePermissionConflict,
	ErrInvalidRole,
	ErrRoleInUse,
	ErrInvalidRolePermission,
	ErrDictKeyExists,
	ErrDictItemValueExists,
	ErrSystemDictDelete,
	ErrSettingKeyExists,
	ErrInvalidSettingValue,
	ErrSystemSettingDelete,
	ErrInvalidLoginMethodSetting,
	MsgLoginMethodDisabled,
	ErrFieldRequired,
	ErrFieldTooLong,
	ErrInvalidEmail,
	ErrInvalidOrder,
	ErrInvalidSearchField,
}

func TestBusinessI18nKeysExistInAllLocales(t *testing.T) {
	locales := []string{"zh", "en", "ja", "ko"}
	for _, locale := range locales {
		t.Run(locale, func(t *testing.T) {
			keys := readI18nKeys(t, locale)
			for _, key := range requiredI18nKeys {
				if !keys[key] {
					t.Fatalf("missing i18n key %q in %s.toml", key, locale)
				}
			}
		})
	}
}

func TestBusinessI18nKeysHaveHTTPStatusMappings(t *testing.T) {
	for _, key := range requiredI18nKeys {
		if _, ok := msgToCode[key]; !ok {
			t.Fatalf("missing HTTP status mapping for i18n key %q", key)
		}
	}
}

func readI18nKeys(t *testing.T, locale string) map[string]bool {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	path := filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "configs", "i18n", locale+".toml")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()

	keyPattern := regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_]*)\s*=`)
	keys := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := keyPattern.FindStringSubmatch(scanner.Text())
		if len(matches) == 2 {
			keys[matches[1]] = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return keys
}
