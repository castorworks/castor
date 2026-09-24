package service

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/redis/go-redis/v9"
)

// 通知事件类型
const (
	// NotificationEventNew 有新通知送达
	NotificationEventNew = "new"
	// NotificationEventChanged 已读状态或可见通知变了：客户端重新取未读数
	NotificationEventChanged = "changed"
)

const (
	// maxStreamsPerUser 每个用户同时打开的实时连接上限（多个标签页）
	maxStreamsPerUser = 5
	// streamBuffer 每个连接缓冲的事件数；满了就丢弃，客户端靠下一次事件的未读数校正
	streamBuffer = 16
)

// NotificationSummary 随"新通知"事件推送的摘要，客户端据此弹出提示
type NotificationSummary struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Level string `json:"level"`
	Link  string `json:"link"`
}

// NotificationEvent 在实例之间（Redis 发布订阅）与推给浏览器的事件
type NotificationEvent struct {
	Kind string `json:"kind"`
	// UserIDs 受影响的用户；为空表示所有人（全局通知、管理员改动）
	UserIDs      []uint               `json:"userIds,omitempty"`
	Notification *NotificationSummary `json:"notification,omitempty"`
}

// Concerns 事件是否与该用户有关
func (e NotificationEvent) Concerns(userID uint) bool {
	return len(e.UserIDs) == 0 || slices.Contains(e.UserIDs, userID)
}

// NotificationStream 通知的实时推送：本实例上的连接订阅，任一实例发布，经 Redis 发布订阅送达所有实例。
type NotificationStream interface {
	// Publish 发布事件（尽力而为：失败只记日志，不影响业务写入）
	Publish(ctx context.Context, event NotificationEvent)
	// Subscribe 为一个连接订阅某用户的事件；连接数超限返回 ErrTooManyRequests。
	// 返回的通道在 Stop 时关闭，调用方用完必须调用 cancel。
	Subscribe(userID uint) (<-chan NotificationEvent, func(), error)
	Start(ctx context.Context) error
	// Stop 停止接收并关闭所有连接的通道，让流式响应结束（HTTP 优雅关闭之前调用）
	Stop()
}

type streamSubscriber struct {
	userID uint
	ch     chan NotificationEvent
}

type notificationStream struct {
	redis   redis.UniversalClient
	channel string

	mu      sync.Mutex
	subs    map[*streamSubscriber]struct{}
	perUser map[uint]int
	stopped bool
	pubsub  *redis.PubSub
	done    chan struct{}
}

// NewNotificationStream 创建实时推送中心
func NewNotificationStream(rdb redis.UniversalClient, ns rediskey.Namespace) NotificationStream {
	return &notificationStream{
		redis:   rdb,
		channel: ns.Key("notifications:events"),
		subs:    map[*streamSubscriber]struct{}{},
		perUser: map[uint]int{},
		done:    make(chan struct{}),
	}
}

func (s *notificationStream) Publish(ctx context.Context, event NotificationEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	// 发布不能跟着请求一起被取消：写库已经成功，推送也该发出去。
	publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := s.redis.Publish(publishCtx, s.channel, payload).Err(); err != nil {
		log.WarnCtx(ctx).Err(err).Msg("Failed to publish notification event")
	}
}

func (s *notificationStream) Subscribe(userID uint) (<-chan NotificationEvent, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return nil, nil, apperror.ErrTooManyRequests
	}
	if s.perUser[userID] >= maxStreamsPerUser {
		return nil, nil, apperror.ErrTooManyRequests
	}
	sub := &streamSubscriber{userID: userID, ch: make(chan NotificationEvent, streamBuffer)}
	s.subs[sub] = struct{}{}
	s.perUser[userID]++
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if _, ok := s.subs[sub]; ok {
				delete(s.subs, sub)
				close(sub.ch)
			}
			if s.perUser[userID]--; s.perUser[userID] <= 0 {
				delete(s.perUser, userID)
			}
		})
	}
	return sub.ch, cancel, nil
}

func (s *notificationStream) Start(ctx context.Context) error {
	pubsub := s.redis.Subscribe(ctx, s.channel)
	// 等订阅确认：之后发布的事件一定收得到。
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return err
	}
	s.mu.Lock()
	s.pubsub = pubsub
	s.mu.Unlock()
	go func() {
		defer close(s.done)
		for msg := range pubsub.Channel() {
			var event NotificationEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}
			s.dispatch(event)
		}
	}()
	return nil
}

// dispatch 把事件交给本实例上相关用户的连接；连接处理不过来时丢弃，不阻塞其他人。
func (s *notificationStream) dispatch(event NotificationEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sub := range s.subs {
		if !event.Concerns(sub.userID) {
			continue
		}
		select {
		case sub.ch <- event:
		default:
		}
	}
}

func (s *notificationStream) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	pubsub := s.pubsub
	for sub := range s.subs {
		delete(s.subs, sub)
		close(sub.ch)
	}
	s.perUser = map[uint]int{}
	s.mu.Unlock()
	if pubsub != nil {
		_ = pubsub.Close()
		<-s.done
	}
}
