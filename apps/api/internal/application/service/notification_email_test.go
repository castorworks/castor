package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/domain/notification"
	gomail "github.com/hyperits/gosuite/net/mail"
)

type memEmailRepo struct {
	mu    sync.Mutex
	items []notification.PendingEmail
}

func (r *memEmailRepo) Enqueue(context.Context, uint) (int64, error) { return 0, nil }

func (r *memEmailRepo) Due(_ context.Context, now time.Time, limit int) ([]notification.PendingEmail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []notification.PendingEmail
	for _, item := range r.items {
		if item.Status == notification.EmailPending && !item.NextAttemptAt.After(now) && len(out) < limit {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *memEmailRepo) find(id uint) *notification.PendingEmail {
	for i := range r.items {
		if r.items[i].ID == id {
			return &r.items[i]
		}
	}
	return nil
}

func (r *memEmailRepo) MarkSent(_ context.Context, id uint, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.find(id)
	item.Status, item.SentAt = notification.EmailSent, &at
	item.Attempts++
	return nil
}

func (r *memEmailRepo) MarkRetry(_ context.Context, id uint, attempts int, lastError string, next time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.find(id)
	item.Attempts, item.LastError, item.NextAttemptAt = attempts, lastError, next
	return nil
}

func (r *memEmailRepo) MarkFinal(_ context.Context, id uint, status notification.EmailStatus, attempts int, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.find(id)
	item.Status, item.Attempts, item.LastError = status, attempts, reason
	return nil
}

func (r *memEmailRepo) Stats(context.Context, uint) (map[notification.EmailStatus]int64, error) {
	return nil, nil
}

func (r *memEmailRepo) StatusOf(context.Context, uint, []uint) (map[uint]notification.EmailStatus, error) {
	return nil, nil
}

type fakeMailer struct {
	mu   sync.Mutex
	sent []*gomail.Message
	fail map[string]error
}

func (m *fakeMailer) Send(_ context.Context, msg *gomail.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.fail[msg.To[0]]; err != nil {
		return err
	}
	m.sent = append(m.sent, msg)
	return nil
}

func (m *fakeMailer) DefaultFrom() string { return "Castor <noreply@castor.test>" }

func pending(id uint, email string, n notification.Notification, wanted bool) notification.PendingEmail {
	return notification.PendingEmail{
		EmailDelivery: notification.EmailDelivery{ID: id, UserID: id, Email: email, Status: notification.EmailPending},
		Notification:  n, RecipientName: "User " + email, StillWanted: wanted,
	}
}

func newEmailFixture(t *testing.T, publicURL string) (*notificationEmailService, *memEmailRepo, *fakeMailer, *time.Time) {
	t.Helper()
	repo := &memEmailRepo{}
	mail := &fakeMailer{fail: map[string]error{}}
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	svc := &notificationEmailService{repo: repo, mail: mail, renderer: testNotificationTemplateRenderer{},
		policy: NotificationMailPolicy{PublicURL: publicURL}, now: func() time.Time { return now }}
	return svc, repo, mail, &now
}

func TestNotificationEmail_SendsSkipsAndRetries(t *testing.T) {
	svc, repo, mail, now := newEmailFixture(t, "https://castor.test")
	past := now.Add(-time.Hour)
	base := notification.Notification{ID: 1, Title: "Maintenance\r\nBcc: victim@evil.test", Content: "Tonight <script>alert(1)</script>\n\nSecond paragraph", Link: "/dashboard/assets"}
	expired := base
	expired.ExpireAt = &past
	repo.items = []notification.PendingEmail{
		pending(1, "alice@castor.test", base, true),
		pending(2, "muted@castor.test", base, false),
		pending(3, "late@castor.test", expired, true),
		pending(4, "bounce@castor.test", base, true),
	}
	mail.fail["bounce@castor.test"] = errors.New("550 mailbox unavailable")

	sent, err := svc.DeliverDue(context.Background())
	if err != nil || sent != 1 {
		t.Fatalf("DeliverDue = %d, %v; want 1 sent", sent, err)
	}
	msg := mail.sent[0]
	if msg.To[0] != "alice@castor.test" || msg.ContentType != gomail.ContentTypeHTML || msg.From != "Castor <noreply@castor.test>" {
		t.Fatalf("message = %+v", msg)
	}
	if msg.Subject != "Maintenance Bcc: victim@evil.test" {
		t.Fatalf("line breaks in the subject must not start new headers: %q", msg.Subject)
	}
	if strings.Contains(msg.Body, "<script>") || !strings.Contains(msg.Body, "&lt;script&gt;") {
		t.Fatal("notification content must be HTML-escaped in the email")
	}
	for _, want := range []string{"Hi User alice@castor.test,", "Second paragraph", `href="https://castor.test/dashboard/assets"`, "Turn these emails off in your profile."} {
		if !strings.Contains(msg.Body, want) {
			t.Errorf("body lacks %q", want)
		}
	}
	status := func(id uint) notification.PendingEmail { return *repo.find(id) }
	if s := status(2); s.Status != notification.EmailSkipped {
		t.Errorf("muted recipient: %+v", s)
	}
	if s := status(3); s.Status != notification.EmailSkipped || !strings.Contains(s.LastError, "expired") {
		t.Errorf("expired notification: %+v", s)
	}
	bounce := status(4)
	if bounce.Status != notification.EmailPending || bounce.Attempts != 1 || !bounce.NextAttemptAt.Equal(now.Add(time.Minute)) || !strings.Contains(bounce.LastError, "550") {
		t.Fatalf("first failure must be retried after a minute: %+v", bounce)
	}

	// 不到下次尝试时间不重发
	if sent, _ := svc.DeliverDue(context.Background()); sent != 0 || status(4).Attempts != 1 {
		t.Fatal("a retry must wait for its backoff")
	}
	// 连续失败：退避递增，第 5 次后放弃
	wantBackoff := []time.Duration{5 * time.Minute, 30 * time.Minute, 2 * time.Hour}
	for i, backoff := range wantBackoff {
		*now = status(4).NextAttemptAt
		svc.DeliverDue(context.Background())
		if s := status(4); s.Attempts != i+2 || !s.NextAttemptAt.Equal(now.Add(backoff)) {
			t.Fatalf("attempt %d: %+v", i+2, s)
		}
	}
	*now = status(4).NextAttemptAt
	svc.DeliverDue(context.Background())
	if s := status(4); s.Status != notification.EmailFailed || s.Attempts != emailMaxAttempts {
		t.Fatalf("after %d attempts the email must fail for good: %+v", emailMaxAttempts, s)
	}
}

func TestNotificationEmail_Links(t *testing.T) {
	svc, _, _, _ := newEmailFixture(t, "https://castor.test/")
	cases := map[string]string{
		"":                           "https://castor.test/dashboard/notifications",
		"/dashboard/users":           "https://castor.test/dashboard/users",
		"https://status.example.com": "https://status.example.com",
		"//evil.test/x":              "https://castor.test/dashboard/notifications",
		"javascript:alert(1)":        "https://castor.test/dashboard/notifications",
	}
	for link, want := range cases {
		if got := svc.absoluteLink(link); got != want {
			t.Errorf("absoluteLink(%q) = %q, want %q", link, got, want)
		}
	}
	noURL, _, _, _ := newEmailFixture(t, "")
	if got := noURL.absoluteLink("/dashboard/users"); got != "" {
		t.Errorf("without PublicURL emails carry no link, got %q", got)
	}
}

func TestNotificationEmail_UnconfiguredMail(t *testing.T) {
	svc := NewNotificationEmailService(&memEmailRepo{}, nil, testNotificationTemplateRenderer{}, NotificationMailPolicy{})
	if svc.Available() {
		t.Fatal("no SMTP client means the channel is unavailable")
	}
	if err := svc.Enqueue(context.Background(), 1); !errors.Is(err, ErrDeliveryChannelNotConfigured) {
		t.Fatalf("Enqueue err = %v", err)
	}
	if got := NewJobCatalog(nil, AssetSweepPolicy{}, nil, svc); len(got) != 1 {
		t.Fatalf("the delivery job must not be registered without SMTP: %d jobs", len(got))
	}
}
