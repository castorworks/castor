package service

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/user"
)

var departmentCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// DepartmentService 部门树管理。写操作受调用者数据范围约束：只能改动范围内的部门，
// 新建根部门（以及把部门移到根）需要全部数据范围。
type DepartmentService interface {
	List(ctx context.Context, scope permission.AccessScope) ([]dto.DepartmentResp, error)
	Save(ctx context.Context, scope permission.AccessScope, d *department.Department) error
	Delete(ctx context.Context, scope permission.AccessScope, id uint) error
}

type departmentService struct {
	repo  department.Repository
	users user.Repository
}

// NewDepartmentService 创建部门服务
func NewDepartmentService(repo department.Repository, users user.Repository) DepartmentService {
	return &departmentService{repo: repo, users: users}
}

func (s *departmentService) List(ctx context.Context, scope permission.AccessScope) ([]dto.DepartmentResp, error) {
	all, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := s.users.CountByDepartment(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.DepartmentResp, len(all))
	for i, d := range all {
		result[i] = dto.DepartmentResp{Department: d, MemberCount: counts[d.ID], InScope: scope.ContainsDepartment(d.ID)}
	}
	return result, nil
}

func (s *departmentService) Save(ctx context.Context, scope permission.AccessScope, d *department.Department) error {
	return s.repo.WithTx(ctx, func(tx department.Repository) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		all, err := tx.List(ctx)
		if err != nil {
			return err
		}
		if err := validateDepartment(d, all); err != nil {
			return err
		}
		if d.ID != 0 && !scope.ContainsDepartment(d.ID) {
			return apperror.ErrDepartmentNotFound
		}
		if d.ParentID == nil {
			if !scope.All {
				return apperror.ErrDataScopeExceedsCaller
			}
		} else if !scope.ContainsDepartment(*d.ParentID) {
			return apperror.ErrDataScopeExceedsCaller
		}
		for _, existing := range all {
			if existing.ID == d.ID {
				d.BaseModel = existing.BaseModel
				break
			}
		}
		return tx.Save(ctx, d)
	})
}

func (s *departmentService) Delete(ctx context.Context, scope permission.AccessScope, id uint) error {
	return s.repo.WithTx(ctx, func(tx department.Repository) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		all, err := tx.List(ctx)
		if err != nil {
			return err
		}
		found := false
		for _, d := range all {
			if d.ID == id {
				found = true
			}
			if d.ParentID != nil && *d.ParentID == id {
				return apperror.ErrDepartmentHasChildren
			}
		}
		if !found || !scope.ContainsDepartment(id) {
			return apperror.ErrDepartmentNotFound
		}
		counts, err := s.users.CountByDepartment(ctx)
		if err != nil {
			return err
		}
		if counts[id] > 0 {
			return apperror.ErrDepartmentHasMembers
		}
		return tx.Delete(ctx, id)
	})
}

// validateDepartment 规范化字段并对整棵树校验：编码唯一、父部门存在、无环、不超过 MaxDepth 层。
func validateDepartment(d *department.Department, all []department.Department) error {
	if d == nil {
		return apperror.ErrInvalidDepartment
	}
	d.Code = strings.TrimSpace(d.Code)
	d.Name = strings.TrimSpace(d.Name)
	if !departmentCodePattern.MatchString(d.Code) || len(d.Code) > 64 ||
		utf8.RuneCountInString(d.Name) < 1 || utf8.RuneCountInString(d.Name) > 100 {
		return apperror.ErrInvalidDepartment
	}
	byID := make(map[uint]department.Department, len(all)+1)
	found := d.ID == 0
	for _, other := range all {
		if other.ID == d.ID {
			found = true
			continue
		}
		if other.Code == d.Code {
			return apperror.ErrDepartmentConflict
		}
		byID[other.ID] = other
	}
	if !found {
		return apperror.ErrDepartmentNotFound
	}
	if d.ParentID == nil {
		return nil
	}
	if *d.ParentID == d.ID {
		return apperror.ErrDepartmentHierarchy
	}
	if _, ok := byID[*d.ParentID]; !ok {
		return apperror.ErrDepartmentHierarchy
	}
	// 以新位置检查整棵树：被移动部门的每个后代都不能因此超深，也不能绕回自己。
	candidate := *d
	if candidate.ID == 0 {
		candidate.ID = ^uint(0)
	}
	byID[candidate.ID] = candidate
	for _, node := range byID {
		depth := 1
		seen := map[uint]bool{node.ID: true}
		for node.ParentID != nil {
			parent, ok := byID[*node.ParentID]
			if !ok || seen[parent.ID] {
				return apperror.ErrDepartmentHierarchy
			}
			seen[parent.ID] = true
			depth++
			if depth > department.MaxDepth {
				return apperror.ErrDepartmentHierarchy
			}
			node = parent
		}
	}
	return nil
}
