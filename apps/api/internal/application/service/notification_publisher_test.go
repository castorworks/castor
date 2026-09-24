package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/castorworks/castor/internal/domain/notification"
)

type testNotificationTemplateRenderer struct{}

func (testNotificationTemplateRenderer) Text(key string, data map[string]any) (string, error) {
	switch key {
	case "NotificationEmailGreeting":
		return fmt.Sprintf("Hi %v,", data["name"]), nil
	case "NotificationEmailOpen":
		return "Open", nil
	case "NotificationEmailFooter":
		return "Turn these emails off in your profile.", nil
	}
	return "", fmt.Errorf("text not found: %s", key)
}

func (testNotificationTemplateRenderer) Render(key string, data map[string]any) (string, string, error) {
	switch key {
	case NotificationTemplateAccountCreated:
		return "Account created", "Your Castor account is ready.", nil
	case NotificationTemplateAssetStatusUpdated:
		return "Asset status updated", fmt.Sprintf("An asset status changed to %v.", data["status"]), nil
	default:
		return "", "", fmt.Errorf("notification template not found: %s", key)
	}
}

func TestNotificationPublisher_PublishToUsers(t *testing.T) {
	notifRepo := newMockNotificationRepo()
	userNotifRepo := newMockUserNotificationRepo(notifRepo)
	publisher := NewNotificationPublisher(newTestNotificationService(notifRepo, userNotifRepo), testNotificationTemplateRenderer{})

	err := publisher.PublishToUsers(context.Background(), NotificationPublishRequest{
		Title:   "Asset approved",
		Content: "Your asset is now active.",
		Type:    string(notification.TypeSystem),
		Level:   string(notification.LevelSuccess),
		Link:    "/dashboard/assets",
		UserIDs: []uint{1, 2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(userNotifRepo.records) != 2 {
		t.Fatalf("expected 2 recipient users, got %d", len(userNotifRepo.records))
	}
}

func TestNotificationPublisher_PublishTemplateToUsers(t *testing.T) {
	notifRepo := newMockNotificationRepo()
	userNotifRepo := newMockUserNotificationRepo(notifRepo)
	publisher := NewNotificationPublisher(newTestNotificationService(notifRepo, userNotifRepo), testNotificationTemplateRenderer{})

	err := publisher.PublishToUsers(context.Background(), NotificationPublishRequest{
		TemplateKey:  NotificationTemplateAssetStatusUpdated,
		TemplateData: map[string]any{"status": "ACTIVE"},
		Type:         string(notification.TypeSystem),
		Level:        string(notification.LevelInfo),
		Link:         "/dashboard/assets",
		UserIDs:      []uint{8},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notifRepo.notifs) != 1 {
		t.Fatalf("expected one notification, got %d", len(notifRepo.notifs))
	}
	for _, item := range notifRepo.notifs {
		if item.Title != "Asset status updated" {
			t.Fatalf("expected rendered title, got %q", item.Title)
		}
		if item.Content != "An asset status changed to ACTIVE." {
			t.Fatalf("expected rendered content, got %q", item.Content)
		}
		if item.TemplateKey != NotificationTemplateAssetStatusUpdated {
			t.Fatalf("expected template key to be stored, got %q", item.TemplateKey)
		}
	}
}
