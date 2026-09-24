package service

import (
	"context"
	"errors"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/query"
)

// SessionService 在线会话应用服务：列出仍有效的授权会话并强制下线。
// 撤销授权会话后，绑定它的访问令牌在下一次请求（含刷新）时即被拒绝。
type SessionService interface {
	// Gets 按数据范围列出在线会话（按会话所属用户）
	Gets(ctx context.Context, scope permission.AccessScope, currentSessionID string, page, size int, order string, opts ...query.Option) ([]dto.SessionResp, int64, error)
	CountActive(ctx context.Context, scope permission.AccessScope) (int64, error)
	// Revoke 撤销会话并返回被撤销的会话；会话属于范围外用户时按不存在处理，
	// 目标用户的权限或数据范围超出调用者时拒绝。
	Revoke(ctx context.Context, scope permission.AccessScope, callerID uint, sessionID string) (*permission.AuthorizationSession, error)
}

type sessionService struct {
	repo  permission.AuthorizationRepository
	rbac  RBACService
	users user.Repository
}

// NewSessionService 创建在线会话应用服务
func NewSessionService(repo permission.AuthorizationRepository, rbac RBACService, users user.Repository) SessionService {
	return &sessionService{repo: repo, rbac: rbac, users: users}
}

func (svc *sessionService) Gets(ctx context.Context, scope permission.AccessScope, currentSessionID string, page, size int, order string, opts ...query.Option) ([]dto.SessionResp, int64, error) {
	items, total, err := svc.repo.ListActiveSessions(ctx, page, size, order, withUserScope(opts, scope, "user_id")...)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]dto.SessionResp, len(items))
	for i := range items {
		resp[i].FromEntity(&items[i], currentSessionID)
	}
	return resp, total, nil
}

func (svc *sessionService) CountActive(ctx context.Context, scope permission.AccessScope) (int64, error) {
	_, total, err := svc.repo.ListActiveSessions(ctx, 1, 1, "", withUserScope(nil, scope, "user_id")...)
	return total, err
}

func (svc *sessionService) Revoke(ctx context.Context, scope permission.AccessScope, callerID uint, sessionID string) (*permission.AuthorizationSession, error) {
	session, err := svc.repo.GetSession(ctx, sessionID)
	if errors.Is(err, shared.ErrNotFound) {
		return nil, apperror.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	// 已撤销或已过期的会话不在在线列表里，对调用者而言等同于不存在。
	if session.RevokedAt != nil || !session.ExpiresAt.After(time.Now()) {
		return nil, apperror.ErrRecordNotFound
	}
	owner, err := svc.users.Get(ctx, session.UserID)
	if errors.Is(err, shared.ErrNotFound) {
		return nil, apperror.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	if !scope.Contains(owner.ID, owner.DepartmentID) {
		return nil, apperror.ErrRecordNotFound
	}
	if callerID == 0 {
		return nil, apperror.ErrTargetUserExceedsCaller
	}
	if err := svc.rbac.EnsureCanManageUser(ctx, callerID, session.UserID); err != nil {
		return nil, err
	}
	if err := svc.repo.RevokeSession(ctx, session.ID); err != nil {
		return nil, err
	}
	return session, nil
}
