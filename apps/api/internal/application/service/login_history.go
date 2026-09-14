package service

import (
	"context"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/pkg/query"
)

// LoginHistoryService 登录历史应用服务接口
type LoginHistoryService interface {
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]dto.LoginHistoryResp, int64, error)
	Record(ctx context.Context, req *dto.LoginHistoryPostReq) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
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

func (svc *loginHistoryService) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]dto.LoginHistoryResp, int64, error) {
	items, total, err := svc.loginHistoryRepo.Gets(ctx, page, size, order, opts...)
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

func (svc *loginHistoryService) Delete(ctx context.Context, id uint) error {
	return svc.loginHistoryRepo.Delete(ctx, id)
}

func (svc *loginHistoryService) BatchDelete(ctx context.Context, ids []uint) error {
	return svc.loginHistoryRepo.BatchDelete(ctx, ids)
}
