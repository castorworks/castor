package service

import (
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/permission"
)

func TestRoleHierarchyClosureSupportsMultipleInheritance(t *testing.T) {
	roles := []permission.Role{
		{ID: 1, Code: "director", IsEnabled: true},
		{ID: 2, Code: "approver", IsEnabled: true},
		{ID: 3, Code: "reviewer", IsEnabled: true},
		{ID: 4, Code: "reader", IsEnabled: true},
	}
	edges := []permission.RoleHierarchy{
		{SeniorRoleID: 1, JuniorRoleID: 2},
		{SeniorRoleID: 1, JuniorRoleID: 3},
		{SeniorRoleID: 2, JuniorRoleID: 4},
		{SeniorRoleID: 3, JuniorRoleID: 4},
	}
	closure := closureSet([]uint{1}, edges, enabledRoleSet(roles))
	for _, id := range []uint{1, 2, 3, 4} {
		if _, ok := closure[id]; !ok {
			t.Fatalf("role %d missing from transitive closure: %#v", id, closure)
		}
	}
	if len(closure) != 4 {
		t.Fatalf("diamond inheritance must de-duplicate roles, got %#v", closure)
	}
}

func TestRoleHierarchyCycleDetection(t *testing.T) {
	acyclic := []permission.RoleHierarchy{{SeniorRoleID: 1, JuniorRoleID: 2}, {SeniorRoleID: 2, JuniorRoleID: 3}}
	if hasHierarchyCycle(acyclic) {
		t.Fatal("acyclic hierarchy reported as cyclic")
	}
	cyclic := append(acyclic, permission.RoleHierarchy{SeniorRoleID: 3, JuniorRoleID: 1})
	if !hasHierarchyCycle(cyclic) {
		t.Fatal("transitive cycle was not detected")
	}
}

func TestSeparationConstraintsIncludeInheritedRoles(t *testing.T) {
	roles := []permission.Role{
		{ID: 1, Code: "senior", IsEnabled: true},
		{ID: 2, Code: "maker", IsEnabled: true},
		{ID: 3, Code: "checker", IsEnabled: true},
	}
	edges := []permission.RoleHierarchy{{SeniorRoleID: 1, JuniorRoleID: 2}}
	constraints := []permission.SeparationConstraint{
		{Type: permission.ConstraintTypeSSD, Cardinality: 2, RoleIDs: []uint{2, 3}, IsEnabled: true},
		{Type: permission.ConstraintTypeDSD, Cardinality: 2, RoleIDs: []uint{2, 3}, IsEnabled: true},
	}
	if !violates([]uint{1, 3}, roles, edges, constraints, permission.ConstraintTypeSSD) {
		t.Fatal("SSD must include inherited maker role")
	}
	if !violates([]uint{1, 3}, roles, edges, constraints, permission.ConstraintTypeDSD) {
		t.Fatal("DSD must include inherited maker role")
	}
	if violates([]uint{1}, roles, edges, constraints, permission.ConstraintTypeDSD) {
		t.Fatal("one matching role must remain below cardinality two")
	}
}

func TestConstraintShapeNormalizationAndValidation(t *testing.T) {
	constraint := &permission.SeparationConstraint{
		Code: " finance ", Name: " Finance duties ", Type: "ssd",
		Cardinality: 2, RoleIDs: []uint{3, 2, 3},
	}
	if err := validateConstraintShape(constraint); err != nil {
		t.Fatalf("valid constraint rejected: %v", err)
	}
	if constraint.Code != "finance" || constraint.Type != permission.ConstraintTypeSSD {
		t.Fatalf("constraint was not normalized: %#v", constraint)
	}
	if len(constraint.RoleIDs) != 2 || constraint.RoleIDs[0] != 2 || constraint.RoleIDs[1] != 3 {
		t.Fatalf("role set was not normalized: %#v", constraint.RoleIDs)
	}
	constraint.Cardinality = 3
	if err := validateConstraintShape(constraint); !errors.Is(err, apperror.ErrInvalidConstraint) {
		t.Fatalf("invalid cardinality returned %v", err)
	}
}
