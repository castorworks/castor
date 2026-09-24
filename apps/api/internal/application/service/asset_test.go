package service

import (
	"bytes"
	"context"
	"errors"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"io"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/query"
)

// ========================
// fakes
// ========================

type fakeAssetStorage struct {
	objects   map[string][]byte
	deleteErr error
}

func newFakeAssetStorage() *fakeAssetStorage {
	return &fakeAssetStorage{objects: make(map[string][]byte)}
}

func (s *fakeAssetStorage) Bucket() string { return "test-bucket" }

func (s *fakeAssetStorage) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.objects[key] = body
	return nil
}

func (s *fakeAssetStorage) Get(_ context.Context, key string, w io.Writer) error {
	body, ok := s.objects[key]
	if !ok {
		return errors.New("object not found")
	}
	_, err := w.Write(body)
	return err
}

func (s *fakeAssetStorage) Delete(_ context.Context, key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.objects, key)
	return nil
}

type fakeAssetRef struct {
	assetID uint
	ref     asset.Reference
	name    string
}

type fakeAssetRepo struct {
	assets map[uint]*asset.Asset
	refs   []fakeAssetRef
	nextID uint
	// createHook 在 Create 落库之前执行，用来模拟并发请求抢先建了同哈希记录
	createHook func() error
	// deleteHook 在删除记录之前执行，用来模拟"检查之后、删除之前"并发登记的引用
	deleteHook func()
	// addReferenceErr 让 AddReference 失败
	addReferenceErr error
}

func newFakeAssetRepo() *fakeAssetRepo {
	return &fakeAssetRepo{assets: make(map[uint]*asset.Asset)}
}

func (r *fakeAssetRepo) add(item asset.Asset) *asset.Asset {
	r.nextID++
	item.ID = r.nextID
	r.assets[item.ID] = &item
	return &item
}

func (r *fakeAssetRepo) Create(_ context.Context, item *asset.Asset) error {
	if r.createHook != nil {
		if err := r.createHook(); err != nil {
			return err
		}
	}
	r.nextID++
	item.ID = r.nextID
	item.UpdatedAt = time.Now()
	cp := *item
	r.assets[item.ID] = &cp
	return nil
}

func (r *fakeAssetRepo) Update(_ context.Context, item *asset.Asset) error {
	cp := *item
	r.assets[item.ID] = &cp
	return nil
}

func (r *fakeAssetRepo) Delete(ctx context.Context, id uint) error {
	return r.BatchDelete(ctx, []uint{id})
}

// BatchDelete 模拟 asset_references 外键：任意一条仍被引用则整体拒绝。
func (r *fakeAssetRepo) BatchDelete(ctx context.Context, ids []uint) error {
	if r.deleteHook != nil {
		r.deleteHook()
	}
	if referenced, _ := r.GetReferencedIDs(ctx, ids); len(referenced) > 0 {
		return errors.New("violates foreign key constraint fk_asset_references_asset")
	}
	for _, id := range ids {
		delete(r.assets, id)
	}
	return nil
}

func (r *fakeAssetRepo) Get(_ context.Context, id uint) (*asset.Asset, error) {
	if item, ok := r.assets[id]; ok {
		cp := *item
		return &cp, nil
	}
	return nil, shared.ErrNotFound
}

func (r *fakeAssetRepo) GetByObjectKey(_ context.Context, objectKey string) (*asset.Asset, error) {
	for _, item := range r.assets {
		if item.ObjectKey == objectKey {
			cp := *item
			return &cp, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (r *fakeAssetRepo) GetByHash(_ context.Context, hash string) (*asset.Asset, error) {
	for _, item := range r.assets {
		if item.Hash == hash {
			cp := *item
			return &cp, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (r *fakeAssetRepo) Gets(context.Context, int, int, string, ...query.Option) ([]asset.Asset, int64, error) {
	return nil, 0, nil
}

func (r *fakeAssetRepo) GetPublicAssets(context.Context, int, int, string) ([]asset.Asset, int64, error) {
	return nil, 0, nil
}

func (r *fakeAssetRepo) GetCombinedStats(context.Context) (map[asset.AssetCategory]int64, map[asset.AssetStatus]int64, int64, error) {
	return nil, nil, 0, nil
}

func (r *fakeAssetRepo) UpdateStatus(_ context.Context, id uint, status asset.AssetStatus) error {
	r.assets[id].Status = status
	return nil
}

func (r *fakeAssetRepo) BatchUpdateStatus(_ context.Context, ids []uint, status asset.AssetStatus) error {
	for _, id := range ids {
		r.assets[id].Status = status
	}
	return nil
}

func (r *fakeAssetRepo) UpdateScope(_ context.Context, id uint, scope asset.AssetScope) error {
	r.assets[id].Scope = scope
	return nil
}

func (r *fakeAssetRepo) UpdateIsPublic(_ context.Context, id uint, isPublic bool) error {
	r.assets[id].IsPublic = isPublic
	return nil
}

func (r *fakeAssetRepo) UpdateFolderPath(_ context.Context, id uint, folderPath string) error {
	r.assets[id].FolderPath = folderPath
	return nil
}

func (r *fakeAssetRepo) IncrementDownloadCount(_ context.Context, id uint) error {
	r.assets[id].DownloadCount++
	return nil
}

func (r *fakeAssetRepo) AddReference(_ context.Context, assetID uint, ref asset.Reference, name string) error {
	if r.addReferenceErr != nil {
		return r.addReferenceErr
	}
	if _, ok := r.assets[assetID]; !ok {
		return errors.New("violates foreign key constraint fk_asset_references_asset")
	}
	for i, existing := range r.refs {
		if existing.assetID == assetID && existing.ref == ref {
			r.refs[i].name = name
			return nil
		}
	}
	r.refs = append(r.refs, fakeAssetRef{assetID, ref, name})
	return nil
}

func (r *fakeAssetRepo) GetReferenceName(_ context.Context, assetID uint, ref asset.Reference) (string, error) {
	for _, existing := range r.refs {
		if existing.assetID == assetID && existing.ref == ref {
			return existing.name, nil
		}
	}
	return "", shared.ErrNotFound
}

func (r *fakeAssetRepo) GetAttached(_ context.Context, field asset.Field, ownerIDs []uint) ([]asset.Attached, error) {
	var attached []asset.Attached
	for _, existing := range r.refs {
		if existing.ref.OwnerType != field.OwnerType || existing.ref.Field != field.Name {
			continue
		}
		for _, id := range ownerIDs {
			if id == existing.ref.OwnerID {
				attached = append(attached, asset.Attached{OwnerID: id, Name: existing.name, Asset: *r.assets[existing.assetID]})
			}
		}
	}
	return attached, nil
}

func (r *fakeAssetRepo) removeRefs(match func(fakeAssetRef) bool) {
	kept := r.refs[:0]
	for _, existing := range r.refs {
		if !match(existing) {
			kept = append(kept, existing)
		}
	}
	r.refs = kept
}

func (r *fakeAssetRepo) RemoveReference(_ context.Context, assetID uint, ref asset.Reference) error {
	r.removeRefs(func(e fakeAssetRef) bool { return e.assetID == assetID && e.ref == ref })
	return nil
}

func (r *fakeAssetRepo) GetReferences(_ context.Context, assetID uint) ([]asset.Reference, error) {
	var refs []asset.Reference
	for _, existing := range r.refs {
		if existing.assetID == assetID {
			refs = append(refs, existing.ref)
		}
	}
	return refs, nil
}

func (r *fakeAssetRepo) CountReferences(ctx context.Context, assetID uint) (int64, error) {
	refs, _ := r.GetReferences(ctx, assetID)
	return int64(len(refs)), nil
}

func (r *fakeAssetRepo) GetReferencedIDs(ctx context.Context, ids []uint) ([]uint, error) {
	var referenced []uint
	for _, id := range ids {
		if count, _ := r.CountReferences(ctx, id); count > 0 {
			referenced = append(referenced, id)
		}
	}
	return referenced, nil
}

func (r *fakeAssetRepo) GetIDsByReference(_ context.Context, ref asset.Reference) ([]uint, error) {
	var ids []uint
	for _, existing := range r.refs {
		if existing.ref == ref {
			ids = append(ids, existing.assetID)
		}
	}
	return ids, nil
}

func (r *fakeAssetRepo) GetIDsByOwner(_ context.Context, ownerType string, ownerID uint) ([]uint, error) {
	var ids []uint
	for _, existing := range r.refs {
		if existing.ref.OwnerType == ownerType && existing.ref.OwnerID == ownerID {
			ids = append(ids, existing.assetID)
		}
	}
	return ids, nil
}

func (r *fakeAssetRepo) GetOrphanAttachments(ctx context.Context, before time.Time, limit int) ([]asset.Asset, error) {
	var orphans []asset.Asset
	for id := uint(1); id <= r.nextID && len(orphans) < limit; id++ {
		item, ok := r.assets[id]
		if !ok || item.Scope != asset.ScopeAttachment || !item.UpdatedAt.Before(before) {
			continue
		}
		if count, _ := r.CountReferences(ctx, id); count == 0 {
			orphans = append(orphans, *item)
		}
	}
	return orphans, nil
}

func (r *fakeAssetRepo) Touch(_ context.Context, id uint) error {
	r.assets[id].UpdatedAt = time.Now()
	return nil
}

func (r *fakeAssetRepo) RemoveOwnerReferences(_ context.Context, ownerType string, ownerID uint) error {
	r.removeRefs(func(e fakeAssetRef) bool { return e.ref.OwnerType == ownerType && e.ref.OwnerID == ownerID })
	return nil
}

type collectingAuditLogService struct {
	mockAuditLogService
	logs []*audit_log.AuditLog
}

func (s *collectingAuditLogService) LogAsync(entry *audit_log.AuditLog) {
	s.logs = append(s.logs, entry)
}

// fakeAssetReferencer 供不关心资产的用户服务测试使用
type fakeAssetReferencer struct {
	attached      []string
	detachedOwner []uint
}

func (f *fakeAssetReferencer) Attach(_ context.Context, objectKey string, _ asset.Reference, _ AttachOptions) error {
	f.attached = append(f.attached, objectKey)
	return nil
}
func (f *fakeAssetReferencer) Detach(context.Context, string, asset.Reference) error  { return nil }
func (f *fakeAssetReferencer) Replace(context.Context, asset.Reference, string) error { return nil }
func (f *fakeAssetReferencer) DetachOwner(_ context.Context, _ string, ownerID uint) error {
	f.detachedOwner = append(f.detachedOwner, ownerID)
	return nil
}
func (f *fakeAssetReferencer) Sync(context.Context, asset.Reference, []asset.AttachedFile, AttachOptions) error {
	return nil
}
func (f *fakeAssetReferencer) ListAttached(context.Context, asset.Field, []uint) (map[uint][]dto.AssetAttachmentResp, error) {
	return nil, nil
}
func (f *fakeAssetReferencer) PrepareAttachedDownload(context.Context, string, asset.Reference) (*AssetDownloadResult, error) {
	return nil, apperror.ErrAssetNotFound
}
func (f *fakeAssetReferencer) WriteContent(context.Context, string, io.Writer) error { return nil }

// pngBytes 是能被 http.DetectContentType 识别为 image/png 的最小内容；
// 追加不同的尾部即可得到哈希不同的"图片"。
func pngBytes(tail string) []byte {
	return append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), tail...)
}

func newAssetTestService(policy AssetPolicy) (*assetService, *fakeAssetRepo, *fakeAssetStorage, *collectingAuditLogService) {
	repo := newFakeAssetRepo()
	storage := newFakeAssetStorage()
	audit := &collectingAuditLogService{}
	return NewAssetService(storage, repo, policy, audit).(*assetService), repo, storage, audit
}

func uploadLibrary(t *testing.T, svc *assetService, filename string, body []byte, isPublic bool) *dto.AssetUploadResp {
	t.Helper()
	resp, err := svc.Upload(context.Background(), &dto.AssetUploadReq{IsPublic: isPublic}, filename,
		fakeMultipartFile{bytes.NewReader(body)}, "application/octet-stream", int64(len(body)))
	if err != nil {
		t.Fatalf("Upload(%s) error = %v", filename, err)
	}
	return resp
}

var testRef = asset.Reference{OwnerType: "article", OwnerID: 1, Field: "cover"}

// ========================
// 上传流水线
// ========================

func TestAssetService_Upload_DeduplicatesByContent(t *testing.T) {
	svc, repo, storage, _ := newAssetTestService(AssetPolicy{})

	first := uploadLibrary(t, svc, "a.png", pngBytes("same"), false)
	second := uploadLibrary(t, svc, "b.png", pngBytes("same"), true)

	if first.IsDuplicate || !second.IsDuplicate {
		t.Fatalf("IsDuplicate = %v, %v; want false, true", first.IsDuplicate, second.IsDuplicate)
	}
	if second.ObjectKey != first.ObjectKey || len(repo.assets) != 1 || len(storage.objects) != 1 {
		t.Fatalf("duplicate content must reuse the existing asset: %d rows, %d objects", len(repo.assets), len(storage.objects))
	}
	if second.IsPublic {
		t.Fatal("a dedup hit must not silently change the existing asset's visibility")
	}
	if first.Scope != asset.ScopeLibrary {
		t.Fatalf("Scope = %q, want LIBRARY", first.Scope)
	}
}

// 哈希唯一索引裁决并发上传：Create 失败后应回读并返回赢家，同时清理自己写入的对象。
func TestAssetService_Upload_ConcurrentDuplicateReturnsWinner(t *testing.T) {
	svc, repo, storage, _ := newAssetTestService(AssetPolicy{})
	body := pngBytes("race")
	hash, _ := calculateFileHash(fakeMultipartFile{bytes.NewReader(body)})
	repo.createHook = func() error {
		repo.createHook = nil
		repo.add(asset.Asset{ObjectKey: "winner.png", Hash: hash, Status: asset.StatusActive, Scope: asset.ScopeLibrary})
		return errors.New("duplicate key value violates unique constraint")
	}

	resp := uploadLibrary(t, svc, "race.png", body, false)
	if !resp.IsDuplicate || resp.ObjectKey != "winner.png" {
		t.Fatalf("got %+v, want the concurrently created asset", resp.AssetResp)
	}
	if len(storage.objects) != 0 {
		t.Fatalf("loser's object must be cleaned up, storage has %d objects", len(storage.objects))
	}
}

// 运营上传的文件去重到一条业务附件上时，必须转入资产库：否则它不出现在资产库列表里，
// 还会随那条业务引用的解除而被回收。
func TestAssetService_Upload_DedupOntoAttachmentJoinsLibrary(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := newAssetTestService(AssetPolicy{})
	avatar := uploadAttachment(t, svc, "shared-logo")
	_ = svc.Attach(ctx, avatar.ObjectKey, testRef, AttachOptions{})

	resp := uploadLibrary(t, svc, "logo.png", pngBytes("shared-logo"), false)
	if !resp.IsDuplicate || resp.ObjectKey != avatar.ObjectKey || resp.Scope != asset.ScopeLibrary {
		t.Fatalf("got %+v, want the same asset promoted to LIBRARY", resp.AssetResp)
	}
	if err := svc.Detach(ctx, avatar.ObjectKey, testRef); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.assets[avatar.ID]; !ok {
		t.Fatal("a file the operator put into the library must survive the business reference")
	}

	// 反过来不成立：业务上传去重到资产库资产上，不得把它降级成会被回收的附件。
	again := uploadAttachment(t, svc, "shared-logo")
	if again.Scope != asset.ScopeLibrary {
		t.Fatalf("Scope = %q, a library asset must never be demoted", again.Scope)
	}
}

// 后台业务表单里的文件字段以 ATTACHMENT 上传，走与资产库相同的全局策略。
func TestAssetService_Upload_ScopeFromRequest(t *testing.T) {
	svc, _, _, _ := newAssetTestService(AssetPolicy{})
	body := pngBytes("form-field")
	resp, err := svc.Upload(context.Background(), &dto.AssetUploadReq{Scope: asset.ScopeAttachment}, "cover.png",
		fakeMultipartFile{bytes.NewReader(body)}, "image/png", int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Scope != asset.ScopeAttachment {
		t.Fatalf("Scope = %q, want ATTACHMENT", resp.Scope)
	}
}

func TestAssetService_Upload_HighRiskNeverPublic(t *testing.T) {
	svc, _, _, _ := newAssetTestService(AssetPolicy{})
	resp := uploadLibrary(t, svc, "logo.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), true)
	if resp.IsPublic {
		t.Fatal("SVG must never be stored as public")
	}
}

func TestAssetService_UploadAttachment_FieldPolicyOnlyNarrows(t *testing.T) {
	global := AssetPolicy{MaxUploadSize: 64, AllowedExtensions: []string{".png", ".pdf"}}
	tests := []struct {
		name     string
		filename string
		body     []byte
		field    AssetPolicy
		wantErr  error
	}{
		{"within both", "a.png", pngBytes("ok"), avatarPolicy, nil},
		{"field forbids extension", "a.pdf", []byte("%PDF-1.4"), avatarPolicy, apperror.ErrAssetInvalidType},
		{"field cannot widen global extensions", "a.gif", []byte("GIF89a"), AssetPolicy{AllowedExtensions: []string{".gif"}}, apperror.ErrAssetInvalidType},
		{"field cannot raise global size", "a.png", pngBytes(string(make([]byte, 80))), AssetPolicy{MaxUploadSize: 1 << 20}, apperror.ErrAssetTooLarge},
		{"field lowers size", "a.png", pngBytes("0123456789"), AssetPolicy{MaxUploadSize: 20}, apperror.ErrAssetTooLarge},
		{"sniffed type must be allowed", "a.png", []byte("plain text pretending"), avatarPolicy, apperror.ErrAssetInvalidType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _ := newAssetTestService(global)
			resp, err := svc.UploadAttachment(context.Background(), AttachmentUpload{
				Filename: tt.filename, File: fakeMultipartFile{bytes.NewReader(tt.body)},
				ContentType: "image/png", Size: int64(len(tt.body)), IsPublic: true, Policy: tt.field,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && (resp.Scope != asset.ScopeAttachment || !resp.IsPublic) {
				t.Fatalf("attachment = %+v, want public ATTACHMENT", resp)
			}
		})
	}
}

// ========================
// 引用保护
// ========================

// 被引用的资产不可删除、下线或取消公开——这是哈希去重得以安全的前提。
func TestAssetService_ReferencedAssetIsProtected(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage, _ := newAssetTestService(AssetPolicy{})
	used := uploadLibrary(t, svc, "used.png", pngBytes("used"), true)
	free := uploadLibrary(t, svc, "free.png", pngBytes("free"), true)
	if err := svc.Attach(ctx, used.ObjectKey, testRef, AttachOptions{}); err != nil {
		t.Fatal(err)
	}

	private := false
	operations := map[string]func() error{
		"Delete":            func() error { return svc.Delete(ctx, used.ID) },
		"BatchDelete":       func() error { return svc.BatchDelete(ctx, []uint{free.ID, used.ID}) },
		"UpdateStatus":      func() error { return svc.UpdateStatus(ctx, used.ID, asset.StatusArchived) },
		"BatchUpdateStatus": func() error { return svc.BatchUpdateStatus(ctx, []uint{free.ID, used.ID}, asset.StatusArchived) },
		"Update unpublish": func() error {
			_, err := svc.Update(ctx, used.ID, &dto.AssetUpdateReq{IsPublic: &private})
			return err
		},
	}
	for name, run := range operations {
		if err := run(); !errors.Is(err, apperror.ErrAssetInUse) {
			t.Errorf("%s: err = %v, want ErrAssetInUse", name, err)
		}
	}
	if len(repo.assets) != 2 || len(storage.objects) != 2 {
		t.Fatalf("a rejected batch must not delete anything: %d rows, %d objects", len(repo.assets), len(storage.objects))
	}
	if got := repo.assets[free.ID].Status; got != asset.StatusActive {
		t.Fatalf("a rejected batch must not change any status, free asset is %s", got)
	}

	if err := svc.Detach(ctx, used.ObjectKey, testRef); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.assets[used.ID]; !ok {
		t.Fatal("a LIBRARY asset must survive losing its last reference")
	}
	if err := svc.Delete(ctx, used.ID); err != nil {
		t.Fatalf("Delete after Detach error = %v", err)
	}
	if _, ok := storage.objects[used.ObjectKey]; ok {
		t.Fatal("deleting an asset must remove its object")
	}
}

// 服务层的先查后删挡不住并发：检查通过之后才登记的引用由外键裁决，此时文件必须原样保留。
func TestAssetService_Delete_LosesRaceAgainstAttach(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage, _ := newAssetTestService(AssetPolicy{})
	item := uploadLibrary(t, svc, "raced.png", pngBytes("raced"), true)
	repo.deleteHook = func() { _ = repo.AddReference(ctx, item.ID, testRef, "") }

	if err := svc.Delete(ctx, item.ID); !errors.Is(err, apperror.ErrAssetInUse) {
		t.Fatalf("err = %v, want ErrAssetInUse", err)
	}
	if _, ok := storage.objects[item.ObjectKey]; !ok {
		t.Fatal("the object must not be removed when the record could not be deleted")
	}
}

// 对象删除失败不回滚已删除的记录：宁可留一个无记录的孤儿对象，也不留一条指向空对象的记录。
func TestAssetService_Delete_StorageFailureDoesNotFail(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage, _ := newAssetTestService(AssetPolicy{})
	item := uploadLibrary(t, svc, "a.png", pngBytes("a"), false)
	storage.deleteErr = errors.New("storage down")

	if err := svc.Delete(ctx, item.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, ok := repo.assets[item.ID]; ok {
		t.Fatal("record must be gone")
	}
}

func TestAssetService_Get_ListsReferences(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := newAssetTestService(AssetPolicy{})
	item := uploadLibrary(t, svc, "a.png", pngBytes("a"), false)
	_ = svc.Attach(ctx, item.ObjectKey, testRef, AttachOptions{})
	_ = svc.Attach(ctx, item.ObjectKey, testRef, AttachOptions{}) // 幂等

	detail, err := svc.Get(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.References) != 1 || detail.References[0] != testRef {
		t.Fatalf("References = %+v, want exactly %+v", detail.References, testRef)
	}
}

func TestAssetService_Attach(t *testing.T) {
	ctx := context.Background()

	t.Run("unknown object key", func(t *testing.T) {
		svc, _, _, _ := newAssetTestService(AssetPolicy{})
		if err := svc.Attach(ctx, "missing.png", testRef, AttachOptions{}); !errors.Is(err, apperror.ErrAssetNotFound) {
			t.Fatalf("err = %v, want ErrAssetNotFound", err)
		}
	})

	t.Run("incomplete reference", func(t *testing.T) {
		svc, _, _, _ := newAssetTestService(AssetPolicy{})
		item := uploadLibrary(t, svc, "a.png", pngBytes("a"), false)
		if err := svc.Attach(ctx, item.ObjectKey, asset.Reference{OwnerType: "article"}, AttachOptions{}); err == nil {
			t.Fatal("an incomplete reference must be rejected")
		}
	})

	t.Run("RequirePublic promotes a private asset and audits it", func(t *testing.T) {
		svc, repo, _, audit := newAssetTestService(AssetPolicy{})
		item := uploadLibrary(t, svc, "a.png", pngBytes("a"), false)
		// 认证中间件放进请求 context 的操作者：审计要记下是谁、从哪里把文件公开的。
		actorCtx := ucontext.WithActor(ctx, ucontext.Actor{UserID: 7, Username: "alice", IP: "203.0.113.9"})
		if err := svc.Attach(actorCtx, item.ObjectKey, testRef, AttachOptions{RequirePublic: true}); err != nil {
			t.Fatal(err)
		}
		if !repo.assets[item.ID].IsPublic {
			t.Fatal("asset must be public after a RequirePublic attach")
		}
		if len(audit.logs) != 1 || audit.logs[0].LogType != audit_log.AuditLogTypeAssetUpdate || audit.logs[0].Target != item.ObjectKey {
			t.Fatalf("promotion must be audited, got %+v", audit.logs)
		}
		if entry := audit.logs[0]; entry.OperatorID != 7 || entry.Operator != "alice" || entry.IpAddr != "203.0.113.9" {
			t.Fatalf("promotion audit must name the operator and IP, got %+v", entry)
		}
	})

	t.Run("RequirePublic rejects high-risk and inactive assets", func(t *testing.T) {
		svc, repo, _, _ := newAssetTestService(AssetPolicy{})
		svg := uploadLibrary(t, svc, "a.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), false)
		archived := uploadLibrary(t, svc, "b.png", pngBytes("b"), true)
		repo.assets[archived.ID].Status = asset.StatusArchived
		for _, key := range []string{svg.ObjectKey, archived.ObjectKey} {
			if err := svc.Attach(ctx, key, testRef, AttachOptions{RequirePublic: true}); !errors.Is(err, apperror.ErrAssetInvalidType) {
				t.Errorf("%s: err = %v, want ErrAssetInvalidType", key, err)
			}
		}
		if len(repo.refs) != 0 || repo.assets[svg.ID].IsPublic {
			t.Fatal("a rejected attach must leave no reference and no visibility change")
		}
	})

	t.Run("RequireCategory", func(t *testing.T) {
		svc, _, _, _ := newAssetTestService(AssetPolicy{})
		pdf := uploadLibrary(t, svc, "a.pdf", []byte("%PDF-1.4 test"), false)
		err := svc.Attach(ctx, pdf.ObjectKey, testRef, AttachOptions{RequireCategory: asset.CategoryImage})
		if !errors.Is(err, apperror.ErrAssetInvalidType) {
			t.Fatalf("err = %v, want ErrAssetInvalidType", err)
		}
	})
}

// ========================
// 附件回收
// ========================

func uploadAttachment(t *testing.T, svc *assetService, tail string) *dto.AssetResp {
	t.Helper()
	body := pngBytes(tail)
	resp, err := svc.UploadAttachment(context.Background(), AttachmentUpload{
		Filename: tail + ".png", File: fakeMultipartFile{bytes.NewReader(body)}, Size: int64(len(body)), IsPublic: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestAssetService_AttachmentIsCollectedWithItsLastReference(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage, _ := newAssetTestService(AssetPolicy{})
	item := uploadAttachment(t, svc, "shared")
	other := asset.Reference{OwnerType: "article", OwnerID: 2, Field: "cover"}
	_ = svc.Attach(ctx, item.ObjectKey, testRef, AttachOptions{})
	_ = svc.Attach(ctx, item.ObjectKey, other, AttachOptions{})

	if err := svc.Detach(ctx, item.ObjectKey, testRef); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.assets[item.ID]; !ok {
		t.Fatal("attachment still referenced by another record must be kept")
	}

	if err := svc.Detach(ctx, item.ObjectKey, other); err != nil {
		t.Fatal(err)
	}
	if len(repo.assets) != 0 || len(storage.objects) != 0 {
		t.Fatalf("orphan attachment must be collected: %d rows, %d objects", len(repo.assets), len(storage.objects))
	}

	// 上传后业务保存失败、从未 Attach 的附件，同样靠 Detach 回收。
	orphan := uploadAttachment(t, svc, "never-attached")
	if err := svc.Detach(ctx, orphan.ObjectKey, testRef); err != nil {
		t.Fatal(err)
	}
	if len(repo.assets) != 0 {
		t.Fatal("never-attached upload must be collected by Detach")
	}
}

// 清扫只回收"超过宽限期、无人引用的业务附件"：刚上传的、被引用的、资产库里的都不动。
func TestAssetService_SweepOrphanAttachments(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage, _ := newAssetTestService(AssetPolicy{})
	age := func(id uint) { repo.assets[id].UpdatedAt = time.Now().Add(-48 * time.Hour) }

	abandoned := uploadAttachment(t, svc, "abandoned")
	fresh := uploadAttachment(t, svc, "fresh")
	inUse := uploadAttachment(t, svc, "in-use")
	_ = svc.Attach(ctx, inUse.ObjectKey, testRef, AttachOptions{})
	library := uploadLibrary(t, svc, "old-lib.png", pngBytes("old-lib"), false)
	for _, id := range []uint{abandoned.ID, inUse.ID, library.ID} {
		age(id)
	}

	if n, err := svc.SweepOrphanAttachments(ctx, 0); n != 0 || err != nil {
		t.Fatalf("a zero TTL must disable sweeping, got %d, %v", n, err)
	}
	n, err := svc.SweepOrphanAttachments(ctx, 24*time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("SweepOrphanAttachments = %d, %v; want 1", n, err)
	}
	if _, ok := repo.assets[abandoned.ID]; ok {
		t.Fatal("the abandoned upload must be collected")
	}
	if _, ok := storage.objects[abandoned.ObjectKey]; ok {
		t.Fatal("the abandoned upload's object must be removed")
	}
	for name, id := range map[string]uint{"fresh": fresh.ID, "in use": inUse.ID, "library": library.ID} {
		if _, ok := repo.assets[id]; !ok {
			t.Errorf("%s asset must be kept", name)
		}
	}

	// 等待清扫的孤儿被新的上传去重复用：宽限期重新计时，调用方来得及 Attach。
	stale := uploadAttachment(t, svc, "reused")
	age(stale.ID)
	again := uploadAttachment(t, svc, "reused")
	if again.ObjectKey != stale.ObjectKey {
		t.Fatal("expected the dedup hit to reuse the stale orphan")
	}
	if n, _ := svc.SweepOrphanAttachments(ctx, 24*time.Hour); n != 0 {
		t.Fatalf("a just-reused attachment must survive the sweep, collected %d", n)
	}

	// 清扫查询之后才登记的引用由外键保护：删不掉就跳过，不能原地打转。
	raced := uploadAttachment(t, svc, "raced-sweep")
	age(raced.ID)
	repo.deleteHook = func() { _ = repo.AddReference(ctx, raced.ID, testRef, "") }
	if n, err := svc.SweepOrphanAttachments(ctx, 24*time.Hour); n != 0 || err != nil {
		t.Fatalf("sweep racing an attach = %d, %v; want 0, nil", n, err)
	}
	if _, ok := storage.objects[raced.ObjectKey]; !ok {
		t.Fatal("an attachment that got attached must keep its object")
	}
}

func TestAssetService_Replace(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := newAssetTestService(AssetPolicy{})
	old := uploadAttachment(t, svc, "old")
	next := uploadAttachment(t, svc, "next")
	_ = svc.Attach(ctx, old.ObjectKey, testRef, AttachOptions{})
	_ = svc.Attach(ctx, next.ObjectKey, testRef, AttachOptions{})

	if err := svc.Replace(ctx, testRef, next.ObjectKey); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.assets[old.ID]; ok {
		t.Fatal("the replaced attachment must be collected")
	}
	if ids, _ := repo.GetIDsByReference(ctx, testRef); len(ids) != 1 || ids[0] != next.ID {
		t.Fatalf("field must reference only the new asset, got %v", ids)
	}

	if err := svc.Replace(ctx, testRef, ""); err != nil {
		t.Fatal(err)
	}
	if len(repo.assets) != 0 {
		t.Fatal("clearing the field must release its attachment")
	}
}

// 多值字段：Sync 之后字段恰好引用给定的文件，被移除的附件随之回收；
// 其中任何一个对象键无效都整体失败，原有引用原样保留。
func TestAssetService_Sync(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := newAssetTestService(AssetPolicy{})
	a := uploadAttachment(t, svc, "a")
	b := uploadAttachment(t, svc, "b")
	c := uploadAttachment(t, svc, "c")
	field := asset.Field{OwnerType: "article", Name: "files"}
	ref := field.Of(1)
	files := func(items ...*dto.AssetResp) []asset.AttachedFile {
		out := make([]asset.AttachedFile, len(items))
		for i, item := range items {
			out[i] = asset.AttachedFile{ObjectKey: item.ObjectKey, Name: "my-" + item.Filename}
		}
		return out
	}

	if err := svc.Sync(ctx, ref, append(files(a, b), files(a)...), AttachOptions{}); err != nil {
		t.Fatal(err)
	}
	listed, err := svc.ListAttached(ctx, field, []uint{1, 2})
	if err != nil || len(listed[1]) != 2 || len(listed[2]) != 0 {
		t.Fatalf("ListAttached = %+v, %v; want two files on owner 1 only", listed, err)
	}
	if listed[1][0].ObjectKey != a.ObjectKey || listed[1][0].Name != "my-a.png" {
		t.Fatalf("first attachment = %+v, want %s named by the reference", listed[1][0], a.ObjectKey)
	}

	if err := svc.Sync(ctx, ref, append(files(b, c), asset.AttachedFile{ObjectKey: "missing.png"}), AttachOptions{}); !errors.Is(err, apperror.ErrAssetNotFound) {
		t.Fatalf("err = %v, want ErrAssetNotFound", err)
	}
	if _, ok := repo.assets[a.ID]; !ok {
		t.Fatal("a failed Sync must not release existing attachments")
	}

	if err := svc.Sync(ctx, ref, files(b, c), AttachOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.assets[a.ID]; ok {
		t.Fatal("an attachment dropped from the field must be collected")
	}
	if listed, _ := svc.ListAttached(ctx, field, []uint{1}); len(listed[1]) != 2 {
		t.Fatalf("after Sync = %+v, want b and c", listed[1])
	}

	if err := svc.Sync(ctx, ref, nil, AttachOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(repo.assets) != 0 {
		t.Fatalf("clearing the field must release everything, %d assets left", len(repo.assets))
	}
}

// 去重让多个上传共用一条资产记录；每个引用各自记名，后来者看不到别人起的文件名。
func TestAssetService_ReferenceNamesAreIsolated(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := newAssetTestService(AssetPolicy{})
	body := pngBytes("same-content")
	upload := func(filename string) *dto.AssetResp {
		resp, err := svc.UploadAttachment(ctx, AttachmentUpload{Filename: filename, File: fakeMultipartFile{bytes.NewReader(body)}, Size: int64(len(body))})
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	field := asset.Field{OwnerType: "notification", Name: "attachments"}
	first := upload("裁员名单.png")
	_ = svc.Attach(ctx, first.ObjectKey, field.Of(1), AttachOptions{DisplayName: "裁员名单.png"})
	second := upload("chart.png")
	if second.ObjectKey != first.ObjectKey {
		t.Fatal("expected the second upload to dedup onto the first")
	}
	_ = svc.Attach(ctx, second.ObjectKey, field.Of(2), AttachOptions{DisplayName: "chart.png"})

	listed, _ := svc.ListAttached(ctx, field, []uint{1, 2})
	if listed[2][0].Name != "chart.png" || listed[1][0].Name != "裁员名单.png" {
		t.Fatalf("names = %q / %q, each reference must keep its own", listed[1][0].Name, listed[2][0].Name)
	}
	result, err := svc.PrepareAttachedDownload(ctx, second.ObjectKey, field.Of(2))
	if err != nil || result.Filename != "chart.png" {
		t.Fatalf("download = %+v, %v; want the reference's own filename", result, err)
	}
}

// 业务模块自己的下载路由：文件必须确实挂在那条业务记录上，私有资产也能下，未挂载一律不存在。
func TestAssetService_PrepareAttachedDownload(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := newAssetTestService(AssetPolicy{})
	field := asset.Field{OwnerType: "notification", Name: "attachments"}
	mine := uploadAttachment(t, svc, "mine")
	_ = svc.Attach(ctx, mine.ObjectKey, field.Of(1), AttachOptions{})
	secret := uploadLibrary(t, svc, "secret.png", pngBytes("secret"), false)

	if _, err := svc.PrepareAttachedDownload(ctx, mine.ObjectKey, field.Of(1)); err != nil {
		t.Fatalf("attached private file: %v", err)
	}
	for name, try := range map[string]struct {
		key string
		ref asset.Reference
	}{
		"file of another record":  {mine.ObjectKey, field.Of(2)},
		"file of another field":   {mine.ObjectKey, asset.Reference{OwnerType: "notification", OwnerID: 1, Field: "cover"}},
		"unattached private file": {secret.ObjectKey, field.Of(1)},
		"missing file":            {"missing.png", field.Of(1)},
	} {
		if _, err := svc.PrepareAttachedDownload(ctx, try.key, try.ref); !errors.Is(err, apperror.ErrAssetNotFound) {
			t.Errorf("%s: err = %v, want ErrAssetNotFound", name, err)
		}
	}
}

func TestAssetService_DetachOwner(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := newAssetTestService(AssetPolicy{})
	cover := uploadAttachment(t, svc, "cover")
	library := uploadLibrary(t, svc, "lib.png", pngBytes("lib"), false)
	_ = svc.Attach(ctx, cover.ObjectKey, testRef, AttachOptions{})
	_ = svc.Attach(ctx, library.ObjectKey, asset.Reference{OwnerType: "article", OwnerID: 1, Field: "banner"}, AttachOptions{})

	if err := svc.DetachOwner(ctx, "article", 1); err != nil {
		t.Fatal(err)
	}
	if len(repo.refs) != 0 {
		t.Fatalf("all references of the owner must be gone, got %+v", repo.refs)
	}
	if _, ok := repo.assets[cover.ID]; ok {
		t.Fatal("owner's attachment must be collected")
	}
	if _, ok := repo.assets[library.ID]; !ok {
		t.Fatal("library asset must be kept")
	}
}

// ========================
// 下载
// ========================

// 免认证下载只放行 IsPublic + ACTIVE + 非高风险类型；其余一律表现为不存在。
func TestAssetService_PrepareDownload_PublicGate(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := newAssetTestService(AssetPolicy{})
	public := uploadLibrary(t, svc, "public.png", pngBytes("public"), true)
	private := uploadLibrary(t, svc, "private.png", pngBytes("private"), false)
	archived := uploadLibrary(t, svc, "archived.png", pngBytes("archived"), true)
	repo.assets[archived.ID].Status = asset.StatusArchived
	risky := repo.add(asset.Asset{ObjectKey: "legacy.svg", Extension: ".svg", IsPublic: true, Status: asset.StatusActive})

	if _, err := svc.PrepareDownload(ctx, public.ObjectKey, true); err != nil {
		t.Fatalf("public asset: %v", err)
	}
	for _, key := range []string{private.ObjectKey, archived.ObjectKey, risky.ObjectKey, "missing"} {
		if _, err := svc.PrepareDownload(ctx, key, true); !errors.Is(err, apperror.ErrAssetNotFound) {
			t.Errorf("%s: err = %v, want ErrAssetNotFound", key, err)
		}
	}
	if _, err := svc.PrepareDownload(ctx, private.ObjectKey, false); err != nil {
		t.Fatalf("admin download of a private asset: %v", err)
	}

	// 业务附件的原始文件名是终端用户的输入，免认证下载只给对象键；后台下载仍给原名。
	avatar := uploadAttachment(t, svc, "zhangsan-id-card")
	if result, _ := svc.PrepareDownload(ctx, avatar.ObjectKey, true); result == nil || result.Filename != avatar.ObjectKey {
		t.Fatalf("public attachment download = %+v, want filename %q", result, avatar.ObjectKey)
	}
	if result, _ := svc.PrepareDownload(ctx, avatar.ObjectKey, false); result == nil || result.Filename != "zhangsan-id-card.png" {
		t.Fatalf("admin attachment download = %+v, want the original filename", result)
	}
	if result, _ := svc.PrepareDownload(ctx, public.ObjectKey, true); result == nil || result.Filename != "public.png" {
		t.Fatalf("public library download = %+v, want the operator-chosen filename", result)
	}

	var buf bytes.Buffer
	if err := svc.WriteContent(ctx, public.ObjectKey, &buf); err != nil || !bytes.Equal(buf.Bytes(), pngBytes("public")) {
		t.Fatalf("WriteContent = %q, %v", buf.Bytes(), err)
	}
}

// ========================
// 头像：资产能力的参考使用者
// ========================

type failingUpdateUserRepo struct {
	*mockUserRepo
	updateErr error
}

func (r *failingUpdateUserRepo) Update(ctx context.Context, item *user.User) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	return r.mockUserRepo.Update(ctx, item)
}

func TestUserAvatarService_Upload(t *testing.T) {
	ctx := context.Background()
	assets, repo, _, _ := newAssetTestService(AssetPolicy{})
	users := &failingUpdateUserRepo{mockUserRepo: newMockUserRepo()}
	users.users["alice"] = &user.User{ID: 7, Username: "alice"}
	svc := NewUserAvatarService(assets, users)
	upload := func(tail string) (*UserAvatarResp, error) {
		body := pngBytes(tail)
		return svc.Upload(ctx, 7, "me.png", fakeMultipartFile{bytes.NewReader(body)}, "image/png", int64(len(body)))
	}

	first, err := upload("first")
	if err != nil {
		t.Fatal(err)
	}
	if users.users["alice"].Avatar != first.ObjectKey {
		t.Fatalf("user avatar = %q, want %q", users.users["alice"].Avatar, first.ObjectKey)
	}
	if refs, _ := repo.GetReferences(ctx, 1); len(refs) != 1 || refs[0] != UserAvatarRef(7) {
		t.Fatalf("avatar must be referenced by the user, got %+v", refs)
	}

	second, err := upload("second")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByObjectKey(ctx, first.ObjectKey); !errors.Is(err, shared.ErrNotFound) {
		t.Fatal("the previous avatar must be collected once replaced")
	}

	// 引用登记失败：刚上传的文件不能成为无人认领的孤儿。
	repo.addReferenceErr = errors.New("db down")
	if _, err := upload("unattached"); err == nil {
		t.Fatal("expected the attach failure to surface")
	}
	repo.addReferenceErr = nil
	if users.users["alice"].Avatar != second.ObjectKey || len(repo.assets) != 1 {
		t.Fatalf("failed attach must leave only the old avatar: avatar=%q, %d assets", users.users["alice"].Avatar, len(repo.assets))
	}

	// 保存用户失败：新上传的文件被回收，旧头像原样保留。
	users.updateErr = errors.New("db down")
	if _, err := upload("third"); err == nil {
		t.Fatal("expected the user update failure to surface")
	}
	if users.users["alice"].Avatar != second.ObjectKey || len(repo.assets) != 1 {
		t.Fatalf("failed change must keep the old avatar only: avatar=%q, %d assets", users.users["alice"].Avatar, len(repo.assets))
	}
	if refs, _ := repo.GetIDsByReference(ctx, UserAvatarRef(7)); len(refs) != 1 {
		t.Fatalf("old avatar reference must survive, got %v", refs)
	}
}

// 管理端在用户表单里选择头像：同样走引用登记，而不是把任意字符串写进 users.avatar。
func TestUserService_Put_AvatarGoesThroughAssetReference(t *testing.T) {
	ctx := managerCtx()
	assets, repo, _, _ := newAssetTestService(AssetPolicy{})
	users := newMockUserRepo()
	users.users["bob"] = &user.User{ID: 9, Username: "bob"}
	svc := &userService{userRepo: users, rbac: &stubRBACService{}, assets: assets}

	picture := uploadLibrary(t, assets, "bob.png", pngBytes("bob"), false)
	document := uploadLibrary(t, assets, "bob.pdf", []byte("%PDF-1.4 bob"), false)

	if _, err := svc.Put(ctx, fullScope, 9, &dto.UserPutReq{Avatar: document.ObjectKey}); !errors.Is(err, apperror.ErrAssetInvalidType) {
		t.Fatalf("non-image avatar: err = %v, want ErrAssetInvalidType", err)
	}
	if _, err := svc.Put(ctx, fullScope, 9, &dto.UserPutReq{Avatar: "https://example.com/a.png"}); !errors.Is(err, apperror.ErrAssetNotFound) {
		t.Fatalf("arbitrary URL avatar: err = %v, want ErrAssetNotFound", err)
	}
	if users.users["bob"].Avatar != "" {
		t.Fatalf("rejected avatar must not be saved, got %q", users.users["bob"].Avatar)
	}

	if _, err := svc.Put(ctx, fullScope, 9, &dto.UserPutReq{Avatar: picture.ObjectKey}); err != nil {
		t.Fatal(err)
	}
	if users.users["bob"].Avatar != picture.ObjectKey || !repo.assets[picture.ID].IsPublic {
		t.Fatal("avatar must be saved and made publicly downloadable")
	}
	if err := assets.Delete(ctx, picture.ID); !errors.Is(err, apperror.ErrAssetInUse) {
		t.Fatalf("an asset used as avatar must not be deletable, err = %v", err)
	}
}
