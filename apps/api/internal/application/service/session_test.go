package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/query"
)

type sessionRepoStub struct {
	permission.AuthorizationRepository
	sessions map[string]*permission.AuthorizationSession
	revoked  []string
}

func (r *sessionRepoStub) GetSession(_ context.Context, id string) (*permission.AuthorizationSession, error) {
	if s, ok := r.sessions[id]; ok {
		clone := *s
		return &clone, nil
	}
	return nil, shared.ErrNotFound
}

func (r *sessionRepoStub) RevokeSession(_ context.Context, id string) error {
	r.revoked = append(r.revoked, id)
	return nil
}

func (r *sessionRepoStub) ListActiveSessions(_ context.Context, _, _ int, _ string, _ ...query.Option) ([]permission.AuthorizationSession, int64, error) {
	var result []permission.AuthorizationSession
	for _, s := range r.sessions {
		result = append(result, *s)
	}
	return result, int64(len(result)), nil
}

type manageGuardRBAC struct {
	RBACService
	allow bool
}

func (r *manageGuardRBAC) EnsureCanManageUser(context.Context, uint, uint) error {
	if r.allow {
		return nil
	}
	return apperror.ErrTargetUserExceedsCaller
}

func TestSessionService_Revoke(t *testing.T) {
	t.Parallel()
	now := time.Now()
	revokedAt := now.Add(-time.Minute)
	newFixture := func(allow bool) (*sessionService, *sessionRepoStub) {
		repo := &sessionRepoStub{sessions: map[string]*permission.AuthorizationSession{
			"live":    {ID: "live", UserID: 5, Username: "boss", ExpiresAt: now.Add(time.Hour)},
			"expired": {ID: "expired", UserID: 5, ExpiresAt: now.Add(-time.Hour)},
			"revoked": {ID: "revoked", UserID: 5, ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt},
		}}
		users := newMockUserRepo()
		users.users["boss"] = &user.User{ID: 5, Username: "boss"}
		return &sessionService{repo: repo, rbac: &manageGuardRBAC{allow: allow}, users: users}, repo
	}

	t.Run("revokes a live session", func(t *testing.T) {
		t.Parallel()
		svc, repo := newFixture(true)
		session, err := svc.Revoke(context.Background(), fullScope, 1, "live")
		if err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}
		if session.Username != "boss" || len(repo.revoked) != 1 || repo.revoked[0] != "live" {
			t.Fatalf("Revoke() = %+v, revoked %v", session, repo.revoked)
		}
	})
	for _, id := range []string{"missing", "expired", "revoked"} {
		t.Run("inactive session "+id+" is not found", func(t *testing.T) {
			t.Parallel()
			svc, repo := newFixture(true)
			if _, err := svc.Revoke(context.Background(), fullScope, 1, id); !errors.Is(err, apperror.ErrRecordNotFound) {
				t.Fatalf("Revoke(%s) error = %v, want ErrRecordNotFound", id, err)
			}
			if len(repo.revoked) != 0 {
				t.Fatal("nothing must be revoked")
			}
		})
	}
	t.Run("more privileged owner cannot be kicked", func(t *testing.T) {
		t.Parallel()
		svc, repo := newFixture(false)
		if _, err := svc.Revoke(context.Background(), fullScope, 1, "live"); !errors.Is(err, apperror.ErrTargetUserExceedsCaller) {
			t.Fatalf("Revoke() error = %v, want ErrTargetUserExceedsCaller", err)
		}
		if len(repo.revoked) != 0 {
			t.Fatal("session must not be revoked when the guard rejects")
		}
	})
	t.Run("unknown caller fails closed", func(t *testing.T) {
		t.Parallel()
		svc, _ := newFixture(true)
		if _, err := svc.Revoke(context.Background(), fullScope, 0, "live"); !errors.Is(err, apperror.ErrTargetUserExceedsCaller) {
			t.Fatalf("Revoke() error = %v, want ErrTargetUserExceedsCaller", err)
		}
	})
}

func TestSessionService_GetsMarksCurrentSession(t *testing.T) {
	t.Parallel()
	repo := &sessionRepoStub{sessions: map[string]*permission.AuthorizationSession{
		"mine":   {ID: "mine", UserID: 1},
		"theirs": {ID: "theirs", UserID: 2},
	}}
	svc := &sessionService{repo: repo}
	items, total, err := svc.Gets(context.Background(), fullScope, "mine", 1, 10, "")
	if err != nil || total != 2 {
		t.Fatalf("Gets() = %v, %d, %v", items, total, err)
	}
	for _, item := range items {
		if item.Current != (item.ID == "mine") {
			t.Fatalf("session %s Current = %v", item.ID, item.Current)
		}
	}
}
