package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

const (
	// streamHeartbeat 心跳间隔：让代理不断开空闲连接，也借机确认会话仍然有效
	streamHeartbeat = 25 * time.Second
	// streamRetryMillis 浏览器断线后的重连间隔
	streamRetryMillis = 5000
)

// NotificationStreamHandler 通知的实时推送（Server-Sent Events）
type NotificationStreamHandler struct {
	stream        service.NotificationStream
	notifications service.NotificationService
	rbac          service.RBACService
}

// NewNotificationStreamHandler 创建实时推送处理器
func NewNotificationStreamHandler(stream service.NotificationStream, notifications service.NotificationService, rbac service.RBACService) *NotificationStreamHandler {
	return &NotificationStreamHandler{stream: stream, notifications: notifications, rbac: rbac}
}

// Stream 当前用户的通知事件流。连接建立时先推一次未读数；之后每有相关变化推
// `unread`（新的未读数），有新通知时先推 `notification`（摘要）。
// 访问令牌到期、会话被撤销或服务停止时结束，浏览器刷新令牌后重连。
// GET /api/v1/account/notifications/stream
func (h *NotificationStreamHandler) Stream(c *gin.Context) {
	userID := ucontext.GetUserID(c)
	sessionID := ucontext.GetAuthorizationSessionID(c)
	events, cancel, err := h.stream.Subscribe(userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	defer cancel()

	// 服务器的 WriteTimeout 是给普通请求的，长连接要单独解除。
	if err := http.NewResponseController(c.Writer).SetWriteDeadline(time.Time{}); err != nil {
		log.Warn(c).Err(err).Msg("Cannot lift the write deadline for the notification stream")
	}
	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	// no-transform：让中间的代理（含 Next.js 的压缩）不缓冲事件。
	header.Set("Cache-Control", "no-cache, no-transform")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	ctx := c.Request.Context()
	write := func(event string, data any) bool {
		payload, err := json.Marshal(data)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, payload); err != nil {
			return false
		}
		c.Writer.Flush()
		return true
	}
	sendUnread := func() bool {
		count, err := h.notifications.GetUnreadCount(ctx, userID)
		if err != nil {
			return false
		}
		return write("unread", dto.NotificationUnreadCountResp{Count: count})
	}

	if _, err := fmt.Fprintf(c.Writer, "retry: %d\n\n", streamRetryMillis); err != nil {
		return
	}
	if !sendUnread() {
		return
	}

	// 连接活不过访问令牌：到期后浏览器重连会走刷新流程，被撤销的会话也就此断开。
	expires := time.NewTimer(tokenLifetime(c))
	defer expires.Stop()
	heartbeat := time.NewTicker(streamHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-expires.C:
			return
		case <-heartbeat.C:
			if _, err := h.rbac.GetSessionAccess(ctx, sessionID, userID); err != nil {
				return
			}
			if _, err := io.WriteString(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Kind == service.NotificationEventNew && event.Notification != nil && !write("notification", event.Notification) {
				return
			}
			if !sendUnread() {
				return
			}
		}
	}
}

// tokenLifetime 本次请求所用访问令牌的剩余有效期（至少 1 秒）
func tokenLifetime(c *gin.Context) time.Duration {
	if exp, ok := jwt.ExtractClaims(c)["exp"].(float64); ok {
		if remaining := time.Until(time.Unix(int64(exp), 0)); remaining > time.Second {
			return remaining
		}
		return time.Second
	}
	return time.Hour
}
