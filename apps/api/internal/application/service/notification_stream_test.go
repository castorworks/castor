package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/pkg/rediskey"
)

func receive(t *testing.T, ch <-chan NotificationEvent) (NotificationEvent, bool) {
	t.Helper()
	select {
	case event, ok := <-ch:
		return event, ok
	case <-time.After(2 * time.Second):
		t.Fatal("no event within 2s")
		return NotificationEvent{}, false
	}
}

func expectNothing(t *testing.T, ch <-chan NotificationEvent) {
	t.Helper()
	select {
	case event := <-ch:
		t.Fatalf("unexpected event %+v", event)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestNotificationStream_EventsCrossReplicasAndReachOnlyTheirUsers(t *testing.T) {
	_, rdb := newJobTestRedis(t)
	ctx := context.Background()
	a := NewNotificationStream(rdb, rediskey.Namespace("test"))
	b := NewNotificationStream(rdb, rediskey.Namespace("test"))
	other := NewNotificationStream(rdb, rediskey.Namespace("another-instance"))
	for _, s := range []NotificationStream{a, b, other} {
		if err := s.Start(ctx); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(s.Stop)
	}

	alice, cancelAlice, err := b.Subscribe(1)
	if err != nil {
		t.Fatal(err)
	}
	defer cancelAlice()
	bob, cancelBob, _ := b.Subscribe(2)
	defer cancelBob()
	foreign, cancelForeign, _ := other.Subscribe(1)
	defer cancelForeign()

	// 在实例 A 发布，实例 B 上的连接收到
	a.Publish(ctx, NotificationEvent{Kind: NotificationEventNew, UserIDs: []uint{1}, Notification: &NotificationSummary{ID: 9, Title: "Hi"}})
	if event, _ := receive(t, alice); event.Kind != NotificationEventNew || event.Notification.Title != "Hi" {
		t.Fatalf("alice got %+v", event)
	}
	expectNothing(t, bob)
	// 另一套部署（不同 Redis 命名空间）的同 ID 用户收不到
	expectNothing(t, foreign)

	// 不带 UserIDs 的事件给所有人
	a.Publish(ctx, NotificationEvent{Kind: NotificationEventChanged})
	receive(t, alice)
	receive(t, bob)
}

func TestNotificationStream_LimitsAndStop(t *testing.T) {
	_, rdb := newJobTestRedis(t)
	s := NewNotificationStream(rdb, rediskey.Namespace("test"))
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	var cancels []func()
	var channels []<-chan NotificationEvent
	for i := 0; i < maxStreamsPerUser; i++ {
		ch, cancel, err := s.Subscribe(7)
		if err != nil {
			t.Fatalf("stream %d: %v", i, err)
		}
		channels, cancels = append(channels, ch), append(cancels, cancel)
	}
	if _, _, err := s.Subscribe(7); !errors.Is(err, apperror.ErrTooManyRequests) {
		t.Fatalf("stream over the per-user cap err = %v", err)
	}
	cancels[0]()
	cancels[0]() // idempotent
	ch, cancel, err := s.Subscribe(7)
	if err != nil {
		t.Fatalf("a slot freed by cancel must be reusable: %v", err)
	}
	defer cancel()

	s.Stop()
	for _, c := range append(channels[1:], ch) {
		if _, ok := <-c; ok {
			t.Fatal("Stop must close every stream")
		}
	}
	if _, _, err := s.Subscribe(8); err == nil {
		t.Fatal("no new streams after Stop")
	}
	for _, cancel := range cancels[1:] {
		cancel() // cancelling after Stop must not panic
	}
}
