package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/notification"
)

type recordingStream struct {
	NotificationStream
	mu     sync.Mutex
	events []NotificationEvent
}

func (r *recordingStream) Publish(_ context.Context, event NotificationEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

type stubEmails struct {
	NotificationEmailService
	available bool
	enqueued  []uint
}

func (s *stubEmails) Available() bool { return s.available }
func (s *stubEmails) Enqueue(_ context.Context, id uint) error {
	s.enqueued = append(s.enqueued, id)
	return nil
}
func (s *stubEmails) Stats(context.Context, uint) (map[notification.EmailStatus]int64, error) {
	return map[notification.EmailStatus]int64{notification.EmailPending: 2, notification.EmailSent: 1}, nil
}

func TestNotificationService_PublishesEventsAndEnqueuesEmails(t *testing.T) {
	ctx := context.Background()
	assets, _, _, _ := newAssetTestService(AssetPolicy{})
	notifRepo := newMockNotificationRepo()
	userNotifRepo := newMockUserNotificationRepo(notifRepo)
	stream := &recordingStream{}
	emails := &stubEmails{available: false}
	svc := NewNotificationService(notifRepo, userNotifRepo, assets, stream, emails)

	if _, err := svc.Post(ctx, &dto.NotificationPostReq{Title: "Mail", UserIDs: []uint{3}, SendEmail: true}); !errors.Is(err, ErrDeliveryChannelNotConfigured) {
		t.Fatalf("sendEmail without SMTP err = %v", err)
	}
	if svc.Channels().Email {
		t.Fatal("channels must report email unavailable")
	}

	emails.available = true
	created, err := svc.Post(ctx, &dto.NotificationPostReq{Title: "Deploy", UserIDs: []uint{3, 4}, SendEmail: true, Level: "WARNING"})
	if err != nil {
		t.Fatal(err)
	}
	if len(emails.enqueued) != 1 || emails.enqueued[0] != created.ID {
		t.Fatalf("emails must be enqueued for the new notification: %v", emails.enqueued)
	}
	if !created.SendEmail || created.EmailStats == nil || created.EmailStats.Pending != 2 || created.EmailStats.Sent != 1 {
		t.Fatalf("response must carry email stats: %+v", created.EmailStats)
	}
	if len(stream.events) != 1 || stream.events[0].Kind != NotificationEventNew || len(stream.events[0].UserIDs) != 2 ||
		stream.events[0].Notification.Title != "Deploy" || stream.events[0].Notification.Level != "WARNING" {
		t.Fatalf("new notification event = %+v", stream.events)
	}

	global, err := svc.Post(ctx, &dto.NotificationPostReq{Title: "Everyone", IsGlobal: true})
	if err != nil {
		t.Fatal(err)
	}
	if last := stream.events[len(stream.events)-1]; len(last.UserIDs) != 0 || last.Notification.ID != global.ID {
		t.Fatalf("a global notification goes to everyone: %+v", last)
	}
	if len(emails.enqueued) != 1 {
		t.Fatal("notifications without sendEmail must not enqueue emails")
	}

	if err := svc.MarkAsRead(ctx, 3, created.ID); err != nil {
		t.Fatal(err)
	}
	if last := stream.events[len(stream.events)-1]; last.Kind != NotificationEventChanged || len(last.UserIDs) != 1 || last.UserIDs[0] != 3 {
		t.Fatalf("reading must refresh only that user's streams: %+v", last)
	}
	if err := svc.Delete(ctx, global.ID); err != nil {
		t.Fatal(err)
	}
	if last := stream.events[len(stream.events)-1]; last.Kind != NotificationEventChanged || len(last.UserIDs) != 0 {
		t.Fatalf("deleting refreshes everyone: %+v", last)
	}
}
