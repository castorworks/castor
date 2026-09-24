package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/notification"
)

const (
	NotificationTemplateAccountCreated     = "NotificationAccountCreated"
	NotificationTemplateAssetStatusUpdated = "NotificationAssetStatusUpdated"
)

type NotificationPublishRequest struct {
	Title        string
	Content      string
	TemplateKey  string
	TemplateData map[string]any
	Type         string
	Level        string
	Link         string
	Extra        string
	UserIDs      []uint
	ExpireAt     *time.Time
}

type NotificationPublisher interface {
	PublishToUsers(ctx context.Context, req NotificationPublishRequest) error
	PublishGlobal(ctx context.Context, req NotificationPublishRequest) error
}

type notificationPublisher struct {
	notificationService NotificationService
	templateRenderer    notification.TemplateRenderer
}

func NewNotificationPublisher(
	notificationService NotificationService,
	templateRenderer notification.TemplateRenderer,
) NotificationPublisher {
	return &notificationPublisher{
		notificationService: notificationService,
		templateRenderer:    templateRenderer,
	}
}

func (p *notificationPublisher) PublishToUsers(ctx context.Context, req NotificationPublishRequest) error {
	postReq, err := p.buildPostReq(req, false)
	if err != nil {
		return err
	}
	_, err = p.notificationService.Post(ctx, postReq)
	return err
}

func (p *notificationPublisher) PublishGlobal(ctx context.Context, req NotificationPublishRequest) error {
	postReq, err := p.buildPostReq(req, true)
	if err != nil {
		return err
	}
	_, err = p.notificationService.Post(ctx, postReq)
	return err
}

func (p *notificationPublisher) buildPostReq(req NotificationPublishRequest, isGlobal bool) (*dto.NotificationPostReq, error) {
	title := req.Title
	content := req.Content
	templateData := ""

	if req.TemplateKey != "" {
		renderedTitle, renderedContent, err := p.templateRenderer.Render(req.TemplateKey, req.TemplateData)
		if err != nil {
			return nil, err
		}
		if title == "" {
			title = renderedTitle
		}
		if content == "" {
			content = renderedContent
		}

		if req.TemplateData != nil {
			data, err := json.Marshal(req.TemplateData)
			if err != nil {
				return nil, err
			}
			templateData = string(data)
		}
	}

	return &dto.NotificationPostReq{
		Title:        title,
		Content:      content,
		TemplateKey:  req.TemplateKey,
		TemplateData: templateData,
		Type:         req.Type,
		Level:        req.Level,
		Link:         req.Link,
		Extra:        req.Extra,
		IsGlobal:     isGlobal,
		UserIDs:      req.UserIDs,
		ExpireAt:     req.ExpireAt,
	}, nil
}
