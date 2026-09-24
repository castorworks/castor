package service

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/pkg/log"
	gomail "github.com/hyperits/gosuite/net/mail"
	smtpmail "github.com/hyperits/gosuite/providers/smtp/mail"
)

const (
	// emailBatchSize 每批发送的邮件数
	emailBatchSize = 100
	// emailMaxAttempts 一封邮件最多尝试次数
	emailMaxAttempts = 5
)

// emailBackoff 第 n 次失败后的等待时间（n 从 1 开始）
var emailBackoff = []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour}

// mailer 发信能力；*smtpmail.Client 满足它
type mailer interface {
	Send(ctx context.Context, msg *gomail.Message) error
	DefaultFrom() string
}

// NotificationMailPolicy 通知邮件的运行策略
type NotificationMailPolicy struct {
	// PublicURL 站点对外地址，用来把通知里的站内链接变成邮件里可点的绝对地址；为空时邮件不带链接
	PublicURL string
}

// NotificationEmailService 通知的邮件通道：登记发件箱、由定时任务逐批发送。
type NotificationEmailService interface {
	// Available 是否配置了邮件服务（没配置时不能要求通知发邮件）
	Available() bool
	// Enqueue 为通知当前的收件人（全局通知为全部用户）登记还没登记过的邮件
	Enqueue(ctx context.Context, notificationID uint) error
	// DeliverDue 发送到期的邮件，返回成功发出的封数（定时任务调用）
	DeliverDue(ctx context.Context) (int64, error)
	Stats(ctx context.Context, notificationID uint) (map[notification.EmailStatus]int64, error)
	StatusOf(ctx context.Context, notificationID uint, userIDs []uint) (map[uint]notification.EmailStatus, error)
}

type notificationEmailService struct {
	repo     notification.EmailRepository
	mail     mailer
	renderer notification.TemplateRenderer
	policy   NotificationMailPolicy
	now      func() time.Time
}

// NewNotificationEmailService 创建通知邮件服务；mailClient 为 nil 表示未配置邮件服务
func NewNotificationEmailService(repo notification.EmailRepository, mailClient *smtpmail.Client, renderer notification.TemplateRenderer, policy NotificationMailPolicy) NotificationEmailService {
	svc := &notificationEmailService{repo: repo, renderer: renderer, policy: policy, now: time.Now}
	if mailClient != nil {
		svc.mail = mailClient
	}
	return svc
}

func (s *notificationEmailService) Available() bool { return s.mail != nil }

func (s *notificationEmailService) Enqueue(ctx context.Context, notificationID uint) error {
	if s.mail == nil {
		return ErrDeliveryChannelNotConfigured
	}
	_, err := s.repo.Enqueue(ctx, notificationID)
	return err
}

func (s *notificationEmailService) Stats(ctx context.Context, notificationID uint) (map[notification.EmailStatus]int64, error) {
	return s.repo.Stats(ctx, notificationID)
}

func (s *notificationEmailService) StatusOf(ctx context.Context, notificationID uint, userIDs []uint) (map[uint]notification.EmailStatus, error) {
	return s.repo.StatusOf(ctx, notificationID, userIDs)
}

func (s *notificationEmailService) DeliverDue(ctx context.Context) (int64, error) {
	if s.mail == nil {
		return 0, ErrDeliveryChannelNotConfigured
	}
	var sent int64
	for {
		batch, err := s.repo.Due(ctx, s.now(), emailBatchSize)
		if err != nil {
			return sent, err
		}
		for _, item := range batch {
			if ctx.Err() != nil {
				return sent, ctx.Err()
			}
			ok, err := s.deliver(ctx, item)
			if err != nil {
				return sent, err
			}
			if ok {
				sent++
			}
		}
		// 本批里的每封都已离开"到期待发"状态（发出、放弃或排到以后），不足一批说明发完了。
		if len(batch) < emailBatchSize {
			return sent, nil
		}
	}
}

// deliver 处理一封：返回是否发出；只有记录状态失败才返回 error。
func (s *notificationEmailService) deliver(ctx context.Context, item notification.PendingEmail) (bool, error) {
	now := s.now()
	switch {
	case !item.StillWanted:
		return false, s.repo.MarkFinal(ctx, item.ID, notification.EmailSkipped, item.Attempts, "the recipient no longer receives notification emails")
	case item.Notification.ExpireAt != nil && !item.Notification.ExpireAt.After(now):
		return false, s.repo.MarkFinal(ctx, item.ID, notification.EmailSkipped, item.Attempts, "the notification expired before it was sent")
	}
	subject, body, err := s.render(item)
	if err == nil {
		err = s.mail.Send(ctx, &gomail.Message{
			From: s.mail.DefaultFrom(), To: []string{item.Email}, Subject: subject, Body: body, ContentType: gomail.ContentTypeHTML,
		})
	}
	if err == nil {
		return true, s.repo.MarkSent(ctx, item.ID, now)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		// 任务被停止：这封留在待发状态，下次再试，不算失败。
		return false, err
	}
	attempts := item.Attempts + 1
	reason := truncateJobMessage(err.Error())
	log.WarnCtx(ctx).Err(err).Uint("emailID", item.ID).Int("attempts", attempts).Msg("Notification email failed")
	if attempts >= emailMaxAttempts {
		return false, s.repo.MarkFinal(ctx, item.ID, notification.EmailFailed, attempts, reason)
	}
	return false, s.repo.MarkRetry(ctx, item.ID, attempts, reason, now.Add(emailBackoff[min(attempts, len(emailBackoff))-1]))
}

var notificationEmailTemplate = template.Must(template.New("email").Parse(`<!doctype html>
<html><body style="margin:0;padding:24px;background:#f5f5f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;color:#1f2937">
<div style="max-width:560px;margin:0 auto;background:#ffffff;border-radius:8px;padding:24px">
<p style="margin:0 0 16px">{{.Greeting}}</p>
<h2 style="margin:0 0 12px;font-size:18px">{{.Title}}</h2>
{{range .Paragraphs}}<p style="margin:0 0 12px;line-height:1.6;white-space:pre-wrap">{{.}}</p>{{end}}
{{if .Link}}<p style="margin:20px 0"><a href="{{.Link}}" style="display:inline-block;background:#111827;color:#ffffff;text-decoration:none;padding:10px 16px;border-radius:6px">{{.Open}}</a></p>{{end}}
<hr style="border:none;border-top:1px solid #e5e7eb;margin:24px 0 12px">
<p style="margin:0;font-size:12px;color:#6b7280">{{.Footer}}</p>
</div></body></html>`))

// render 生成主题与 HTML 正文。html/template 转义标题与正文，通知内容不能向邮件注入标记。
func (s *notificationEmailService) render(item notification.PendingEmail) (string, string, error) {
	greeting, err := s.renderer.Text("NotificationEmailGreeting", map[string]any{"name": item.RecipientName})
	if err != nil {
		return "", "", err
	}
	open, err := s.renderer.Text("NotificationEmailOpen", nil)
	if err != nil {
		return "", "", err
	}
	footer, err := s.renderer.Text("NotificationEmailFooter", nil)
	if err != nil {
		return "", "", err
	}
	var paragraphs []string
	for _, p := range strings.Split(strings.ReplaceAll(item.Notification.Content, "\r\n", "\n"), "\n\n") {
		if strings.TrimSpace(p) != "" {
			paragraphs = append(paragraphs, strings.TrimSpace(p))
		}
	}
	var buf bytes.Buffer
	if err := notificationEmailTemplate.Execute(&buf, map[string]any{
		"Greeting": greeting, "Title": item.Notification.Title, "Paragraphs": paragraphs,
		"Link": s.absoluteLink(item.Notification.Link), "Open": open, "Footer": footer,
	}); err != nil {
		return "", "", err
	}
	// 主题里的换行会被当成邮件头的分隔：去掉。
	subject := strings.Join(strings.Fields(item.Notification.Title), " ")
	return subject, buf.String(), nil
}

// absoluteLink 邮件里的链接必须是绝对地址：站内路径接到 PublicURL 上；没配 PublicURL 时不放链接。
func (s *notificationEmailService) absoluteLink(link string) string {
	base := strings.TrimRight(s.policy.PublicURL, "/")
	if base == "" {
		return ""
	}
	if link == "" {
		return base + "/dashboard/notifications"
	}
	if strings.HasPrefix(link, "/") && !strings.HasPrefix(link, "//") {
		return base + link
	}
	if u, err := url.Parse(link); err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" {
		return link
	}
	return base + "/dashboard/notifications"
}
