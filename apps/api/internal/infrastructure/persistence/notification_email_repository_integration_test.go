package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
)

func TestNotificationEmailRepository(t *testing.T) {
	db := openMFATestDB(t)
	if err := db.AutoMigrate(&models.NotificationModel{}, &models.UserNotificationModel{}, &models.NotificationEmailModel{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE TABLE notification_emails, user_notifications, notifications RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	mk := func(name, email string, verified, enabled, muted bool) uint {
		u := models.UserModel{Username: name, AccountSource: "INTERNAL", Email: email, EmailVerified: verified, Enable: enabled, MuteNotificationEmails: muted}
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		return u.ID
	}
	ok := mk("mail-ok", "ok@castor.test", true, true, false)
	unverified := mk("mail-unverified", "unverified@castor.test", false, true, false)
	muted := mk("mail-muted", "muted@castor.test", true, true, true)
	disabled := mk("mail-disabled", "disabled@castor.test", true, false, false)
	noEmail := mk("mail-none", "", true, true, false)
	other := mk("mail-other", "other@castor.test", true, true, false)

	targeted := models.NotificationModel{Title: "Targeted", Content: "Body", Link: "/dashboard", SendEmail: true}
	global := models.NotificationModel{Title: "Global", IsGlobal: true, SendEmail: true}
	quiet := models.NotificationModel{Title: "In-app only", IsGlobal: true}
	for _, n := range []*models.NotificationModel{&targeted, &global, &quiet} {
		if err := db.Create(n).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, uid := range []uint{ok, unverified, muted, disabled, noEmail} {
		if err := db.Create(&models.UserNotificationModel{UserID: uid, NotificationID: targeted.ID}).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewNotificationEmailRepository(db)

	if n, err := repo.Enqueue(ctx, targeted.ID); err != nil || n != 1 {
		t.Fatalf("targeted Enqueue = %d, %v; only the verified, enabled, unmuted recipient qualifies", n, err)
	}
	if n, _ := repo.Enqueue(ctx, targeted.ID); n != 0 {
		t.Fatal("enqueueing again must not duplicate emails")
	}
	if n, err := repo.Enqueue(ctx, global.ID); err != nil || n != 2 {
		t.Fatalf("global Enqueue = %d, %v; want ok + other", n, err)
	}
	if n, _ := repo.Enqueue(ctx, quiet.ID); n != 0 {
		t.Fatal("notifications without sendEmail enqueue nothing")
	}

	// 登记之后用户关闭了通知邮件：发送时要看出来
	if err := db.Model(&models.UserModel{}).Where("id = ?", other).Update("mute_notification_emails", true).Error; err != nil {
		t.Fatal(err)
	}
	due, err := repo.Due(ctx, time.Now().Add(time.Second), 10)
	if err != nil || len(due) != 3 {
		t.Fatalf("Due = %d, %v", len(due), err)
	}
	byUser := map[uint]notification.PendingEmail{}
	for _, d := range due {
		byUser[d.UserID*1000+d.NotificationID] = d
	}
	first := byUser[ok*1000+targeted.ID]
	if first.Email != "ok@castor.test" || first.Notification.Title != "Targeted" || first.Notification.Link != "/dashboard" || !first.StillWanted || first.RecipientName != "mail-ok" {
		t.Fatalf("pending email = %+v", first)
	}
	if byUser[other*1000+global.ID].StillWanted {
		t.Fatal("a recipient who muted notification emails after enqueueing is no longer wanted")
	}

	if err := repo.MarkSent(ctx, first.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkRetry(ctx, byUser[ok*1000+global.ID].ID, 1, "421 try later", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkFinal(ctx, byUser[other*1000+global.ID].ID, notification.EmailSkipped, 0, "muted"); err != nil {
		t.Fatal(err)
	}
	if due, _ := repo.Due(ctx, time.Now(), 10); len(due) != 0 {
		t.Fatalf("nothing is due after sending, retrying later and skipping: %+v", due)
	}
	stats, _ := repo.Stats(ctx, global.ID)
	if stats[notification.EmailPending] != 1 || stats[notification.EmailSkipped] != 1 {
		t.Fatalf("Stats = %v", stats)
	}
	statuses, _ := repo.StatusOf(ctx, targeted.ID, []uint{ok, muted})
	if statuses[ok] != notification.EmailSent || statuses[muted] != "" {
		t.Fatalf("StatusOf = %v", statuses)
	}

	if err := db.Delete(&models.NotificationModel{}, global.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stats, _ := repo.Stats(ctx, global.ID); len(stats) != 0 {
		t.Fatalf("emails must be deleted with their notification: %v", stats)
	}
}
