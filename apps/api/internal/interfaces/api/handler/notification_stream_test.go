package handler

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/gin-gonic/gin"
)

type fakeStream struct {
	service.NotificationStream
	ch chan service.NotificationEvent
}

func (f *fakeStream) Subscribe(uint) (<-chan service.NotificationEvent, func(), error) {
	return f.ch, func() {}, nil
}

type countingNotifications struct {
	service.NotificationService
	count atomic.Int64
}

func (n *countingNotifications) GetUnreadCount(context.Context, uint) (int64, error) {
	return n.count.Load(), nil
}

type sessionRBAC struct {
	service.RBACService
	revoked atomic.Bool
}

func (r *sessionRBAC) GetSessionAccess(context.Context, string, uint) (*service.AccessSnapshot, error) {
	if r.revoked.Load() {
		return nil, errors.New("revoked")
	}
	return &service.AccessSnapshot{}, nil
}

func startStreamServer(t *testing.T, h *NotificationStreamHandler, exp time.Time) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/stream", func(c *gin.Context) {
		c.Set("JWT_PAYLOAD", jwt.MapClaims{constant.JWT_IDENTITY_KEY: float64(3), constant.JWT_AUTHORIZATION_SESSION: "s-1", "exp": float64(exp.Unix())})
	}, h.Stream)
	server := httptest.NewUnstartedServer(engine)
	// 普通请求的写超时很短：流必须不受它影响。
	server.Config.WriteTimeout = 300 * time.Millisecond
	server.Start()
	t.Cleanup(server.Close)
	return server.URL + "/stream"
}

func readEvents(t *testing.T, url string) <-chan string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("Content-Type") != "text/event-stream; charset=utf-8" || !strings.Contains(resp.Header.Get("Cache-Control"), "no-transform") {
		t.Fatalf("headers = %v", resp.Header)
	}
	lines := make(chan string, 64)
	go func() {
		defer resp.Body.Close()
		defer close(lines)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if line := scanner.Text(); line != "" {
				lines <- line
			}
		}
	}()
	return lines
}

func nextLine(t *testing.T, lines <-chan string) string {
	t.Helper()
	select {
	case line, ok := <-lines:
		if !ok {
			return "<closed>"
		}
		return line
	case <-time.After(3 * time.Second):
		t.Fatal("no line within 3s")
		return ""
	}
}

func TestNotificationStream_PushesUnreadAndNewNotifications(t *testing.T) {
	stream := &fakeStream{ch: make(chan service.NotificationEvent, 4)}
	notifications := &countingNotifications{}
	notifications.count.Store(2)
	h := NewNotificationStreamHandler(stream, notifications, &sessionRBAC{})
	lines := readEvents(t, startStreamServer(t, h, time.Now().Add(time.Hour)))

	for _, want := range []string{"retry: 5000", "event: unread", `data: {"count":2}`} {
		if got := nextLine(t, lines); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
	// 超过服务器的 WriteTimeout 之后连接仍然可用
	time.Sleep(500 * time.Millisecond)
	notifications.count.Store(3)
	stream.ch <- service.NotificationEvent{Kind: service.NotificationEventNew, Notification: &service.NotificationSummary{ID: 5, Title: "Deploy", Level: "INFO"}}
	for _, want := range []string{"event: notification", `data: {"id":5,"title":"Deploy","type":"","level":"INFO","link":""}`, "event: unread", `data: {"count":3}`} {
		if got := nextLine(t, lines); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
	// 服务停止（通道关闭）时流结束
	close(stream.ch)
	if got := nextLine(t, lines); got != "<closed>" {
		t.Fatalf("stream should end when the hub stops, got %q", got)
	}
}

func TestNotificationStream_EndsWhenTheTokenExpires(t *testing.T) {
	h := NewNotificationStreamHandler(&fakeStream{ch: make(chan service.NotificationEvent)}, &countingNotifications{}, &sessionRBAC{})
	lines := readEvents(t, startStreamServer(t, h, time.Now().Add(1500*time.Millisecond)))
	nextLine(t, lines)
	nextLine(t, lines)
	nextLine(t, lines)
	if got := nextLine(t, lines); got != "<closed>" {
		t.Fatalf("stream should end at token expiry, got %q", got)
	}
}
