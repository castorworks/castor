package department

import "github.com/castorworks/castor/internal/domain/shared"

// MaxDepth 是部门树的最大层数（根部门为第 1 层）。
const MaxDepth = 8

// Department 组织树上的一个部门。名称与编码属于组织自身的数据（和角色名一样），
// 不走四语言文案。每个用户最多归属一个部门，数据范围据此划定可见的用户。
type Department struct {
	shared.BaseModel
	ParentID  *uint  `json:"parentId"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
	IsEnabled bool   `json:"isEnabled"`
}

// Subtree 返回 rootID 及其全部后代的 ID（含 rootID 本身）。部门树规模有限，
// 在内存中展开即可，数据库里不必维护物化路径，移动部门也不需要级联改写。
func Subtree(all []Department, rootID uint) []uint {
	children := make(map[uint][]uint, len(all))
	exists := false
	for _, d := range all {
		if d.ID == rootID {
			exists = true
		}
		if d.ParentID != nil {
			children[*d.ParentID] = append(children[*d.ParentID], d.ID)
		}
	}
	if !exists {
		return nil
	}
	result := []uint{rootID}
	seen := map[uint]bool{rootID: true}
	for i := 0; i < len(result); i++ {
		for _, child := range children[result[i]] {
			if !seen[child] {
				seen[child] = true
				result = append(result, child)
			}
		}
	}
	return result
}
