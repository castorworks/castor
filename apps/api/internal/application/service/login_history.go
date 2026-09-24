package service

import (
	"context"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/query"
)

// LoginHistoryService 登录历史应用服务接口
type LoginHistoryService interface {
	// Gets 按数据范围列出登录记录（按记录所属用户）；用户查看自己的记录时传全部范围并附加用户名条件
	Gets(ctx context.Context, scope permission.AccessScope, page, size int, order string, opts ...query.Option) ([]dto.LoginHistoryResp, int64, error)
	Record(ctx context.Context, req *dto.LoginHistoryPostReq) error
	// Delete / BatchDelete 只删得到范围内的记录；有任何一条在范围外（或不存在）就整体按不存在拒绝
	Delete(ctx context.Context, scope permission.AccessScope, id uint) error
	BatchDelete(ctx context.Context, scope permission.AccessScope, ids []uint) error
}

type loginHistoryService struct {
	loginHistoryRepo login_history.Repository
}

// NewLoginHistoryService 创建登录历史应用服务
func NewLoginHistoryService(loginHistoryRepo login_history.Repository) LoginHistoryService {
	return &loginHistoryService{
		loginHistoryRepo: loginHistoryRepo,
	}
}

func (svc *loginHistoryService) Gets(ctx context.Context, scope permission.AccessScope, page, size int, order string, opts ...query.Option) ([]dto.LoginHistoryResp, int64, error) {
	items, total, err := svc.loginHistoryRepo.Gets(ctx, page, size, order, withUserScope(opts, scope, "user_id")...)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.LoginHistoryResp, len(items))
	for i, item := range items {
		if err := resp[i].FromEntity(&item); err != nil {
			return nil, 0, err
		}
	}
	return resp, total, nil
}

func (svc *loginHistoryService) Record(ctx context.Context, req *dto.LoginHistoryPostReq) error {
	item := &login_history.LoginHistory{
		UserID:      req.UserID,
		Username:    req.Username,
		IpAddr:      req.IpAddr,
		UserAgent:   req.UserAgent,
		LoginMethod: req.LoginMethod,
		Success:     req.Success,
	}

	return svc.loginHistoryRepo.Create(ctx, item)
}

func (svc *loginHistoryService) Delete(ctx context.Context, scope permission.AccessScope, id uint) error {
	if err := svc.ensureInScope(ctx, scope, []uint{id}); err != nil {
		return err
	}
	return svc.loginHistoryRepo.Delete(ctx, id)
}

// ensureInScope 确认 ids 全部存在且都在数据范围内。
func (svc *loginHistoryService) ensureInScope(ctx context.Context, scope permission.AccessScope, ids []uint) error {
	unique := permission.NormalizeIDs(ids)
	if scope.All || len(unique) == 0 {
		return nil
	}
	opts := withUserScope([]query.Option{*query.NewOption("id IN ?", unique)}, scope, "user_id")
	_, total, err := svc.loginHistoryRepo.Gets(ctx, 1, 1, "", opts...)
	if err != nil {
		return err
	}
	if total != int64(len(unique)) {
		return shared.ErrNotFound
	}
	return nil
}

func (svc *loginHistoryService) BatchDelete(ctx context.Context, scope permission.AccessScope, ids []uint) error {
	if err := svc.ensureInScope(ctx, scope, ids); err != nil {
		return err
	}
	return svc.loginHistoryRepo.BatchDelete(ctx, ids)
}
