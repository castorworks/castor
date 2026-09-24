package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/notification"
)

// newTestNotificationService 不关心附件的通知测试使用：资产侧用真实服务 + 内存仓储。
func newTestNotificationService(notifRepo notification.NotificationRepository, userNotifRepo notification.UserNotificationRepository) NotificationService {
	assets, _, _, _ := newAssetTestService(AssetPolicy{})
	return NewNotificationService(notifRepo, userNotifRepo, assets, nil, nil)
}

// uploadPrivateAttachment 与通知模块的上传路由一致：私有业务附件
func uploadPrivateAttachment(t *testing.T, svc *assetService, name string) *dto.AssetResp {
	t.Helper()
	body := pngBytes(name)
	resp, err := svc.UploadAttachment(context.Background(), AttachmentUpload{
		Filename: name + ".png", File: fakeMultipartFile{bytes.NewReader(body)}, Size: int64(len(body)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func attachmentReqs(items ...*dto.AssetResp) []dto.NotificationAttachmentReq {
	reqs := make([]dto.NotificationAttachmentReq, len(items))
	for i, item := range items {
		reqs[i] = dto.NotificationAttachmentReq{ObjectKey: item.ObjectKey, Name: item.Filename}
	}
	return reqs
}

// 通知附件是资产能力的第二个使用者：多值字段、私有文件、由通知模块自己的路由鉴权下载。
func TestNotificationService_Attachments(t *testing.T) {
	ctx := context.Background()
	assets, assetRepo, _, _ := newAssetTestService(AssetPolicy{})
	notifRepo := newMockNotificationRepo()
	userNotifRepo := newMockUserNotificationRepo(notifRepo)
	svc := NewNotificationService(notifRepo, userNotifRepo, assets, nil, nil)

	report := uploadPrivateAttachment(t, assets, "report")
	chart := uploadPrivateAttachment(t, assets, "chart")
	const recipient, stranger = uint(7), uint(8)

	created, err := svc.Post(ctx, &dto.NotificationPostReq{
		Title: "Q3", UserIDs: []uint{recipient}, Attachments: attachmentReqs(report, chart),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Attachments) != 2 || created.Attachments[0].Name != "report.png" {
		t.Fatalf("created attachments = %+v", created.Attachments)
	}
	if assetRepo.assets[report.ID].IsPublic {
		t.Fatal("notification attachments must stay private")
	}

	// 收件人的列表里带附件；下载需要「通知可见 + 文件确属该通知」。
	inbox, _, err := svc.GetUserNotifications(ctx, recipient, 1, 10, false)
	if err != nil || len(inbox) != 1 || len(inbox[0].Attachments) != 2 {
		t.Fatalf("inbox = %+v, %v", inbox, err)
	}
	if result, err := svc.PrepareAttachmentDownload(ctx, recipient, created.ID, report.ObjectKey); err != nil || result.Filename != "report.png" {
		t.Fatalf("recipient download = %+v, %v", result, err)
	}
	if _, err := svc.PrepareAttachmentDownload(ctx, stranger, created.ID, report.ObjectKey); !errors.Is(err, apperror.ErrNotificationNotDelivered) {
		t.Fatalf("stranger download err = %v, want ErrNotificationNotDelivered", err)
	}
	other := uploadPrivateAttachment(t, assets, "other-module-file")
	if _, err := svc.PrepareAttachmentDownload(ctx, recipient, created.ID, other.ObjectKey); !errors.Is(err, apperror.ErrAssetNotFound) {
		t.Fatalf("foreign file via own notification err = %v, want ErrAssetNotFound", err)
	}

	// 管理端下载不看投递对象，但同样只认挂在这条通知上的文件。
	if result, err := svc.PrepareAdminAttachmentDownload(ctx, created.ID, report.ObjectKey); err != nil || result.Filename != "report.png" {
		t.Fatalf("admin download = %+v, %v", result, err)
	}
	if _, err := svc.PrepareAdminAttachmentDownload(ctx, created.ID, other.ObjectKey); !errors.Is(err, apperror.ErrAssetNotFound) {
		t.Fatalf("admin download of a foreign file err = %v, want ErrAssetNotFound", err)
	}
	if _, err := svc.PrepareAdminAttachmentDownload(ctx, created.ID+100, report.ObjectKey); err == nil {
		t.Fatal("admin download through a missing notification must fail")
	}

	// 更新：nil 不动附件；给出新集合则同步，被移除的附件回收。
	if _, err := svc.Put(ctx, created.ID, &dto.NotificationPutReq{Title: "Q3 v2"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := svc.Get(ctx, created.ID); len(got.Attachments) != 2 {
		t.Fatalf("a Put without attachments must keep them, got %+v", got.Attachments)
	}
	onlyChart := attachmentReqs(chart)
	updated, err := svc.Put(ctx, created.ID, &dto.NotificationPutReq{Title: "Q3 v3", Attachments: &onlyChart})
	if err != nil || len(updated.Attachments) != 1 || updated.Attachments[0].ObjectKey != chart.ObjectKey {
		t.Fatalf("updated attachments = %+v, %v", updated, err)
	}
	if _, ok := assetRepo.assets[report.ID]; ok {
		t.Fatal("an attachment removed from the notification must be collected")
	}

	// 被通知引用的文件受保护；删除通知后释放并回收。
	if err := assets.Delete(ctx, chart.ID); !errors.Is(err, apperror.ErrAssetInUse) {
		t.Fatalf("deleting an attached file err = %v, want ErrAssetInUse", err)
	}
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := assetRepo.assets[chart.ID]; ok {
		t.Fatal("deleting the notification must release its attachments")
	}
}

// 附件对象键无效时，通知不能已经投递出去。
func TestNotificationService_Post_InvalidAttachmentRollsBack(t *testing.T) {
	ctx := context.Background()
	assets, assetRepo, _, _ := newAssetTestService(AssetPolicy{})
	notifRepo := newMockNotificationRepo()
	userNotifRepo := newMockUserNotificationRepo(notifRepo)
	svc := NewNotificationService(notifRepo, userNotifRepo, assets, nil, nil)
	valid := uploadPrivateAttachment(t, assets, "valid")

	_, err := svc.Post(ctx, &dto.NotificationPostReq{
		Title: "broken", UserIDs: []uint{7},
		Attachments: append(attachmentReqs(valid), dto.NotificationAttachmentReq{ObjectKey: "missing.png"}),
	})
	if !errors.Is(err, apperror.ErrAssetNotFound) {
		t.Fatalf("err = %v, want ErrAssetNotFound", err)
	}
	if inbox, _, _ := svc.GetUserNotifications(ctx, 7, 1, 10, false); len(inbox) != 0 {
		t.Fatalf("a rolled-back notification must not reach the inbox: %+v", inbox)
	}
	if len(assetRepo.refs) != 0 {
		t.Fatalf("no reference may survive the rollback: %+v", assetRepo.refs)
	}
}
