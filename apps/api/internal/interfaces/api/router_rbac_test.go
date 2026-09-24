package api

import (
	"testing"

	"github.com/castorworks/castor/internal/domain/permission"
)

func TestRBACResourcesCoverProtectedRoutes(t *testing.T) {
	declared := make(map[string]struct{})
	for _, route := range parseRoutes(t) {
		if route.auth == authAdmin {
			declared[route.path+"\x00"+route.method] = struct{}{}
		}
	}
	configured := make(map[string]struct{})
	for _, resource := range permission.DefaultResources {
		for _, action := range resource.Actions {
			configured[resource.Path+"\x00"+action] = struct{}{}
		}
	}
	missing, stale := setDifference(declared, configured), setDifference(configured, declared)
	if len(missing) != 0 || len(stale) != 0 {
		t.Fatalf("RBAC route catalog drifted; missing=%v stale=%v", missing, stale)
	}
}
