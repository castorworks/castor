package permission

import "time"

// ConstraintType identifies the two separation-of-duty relations defined by RBAC3.
type ConstraintType string

const (
	ConstraintTypeSSD ConstraintType = "SSD"
	ConstraintTypeDSD ConstraintType = "DSD"
)

// RolePermission is one atomic permission assignment (PA): a role may perform
// one action on one resource.
type RolePermission struct {
	ID         uint      `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  uint      `json:"createdBy"`
	RoleID     uint      `json:"roleId"`
	ResourceID uint      `json:"resourceId"`
	Action     string    `json:"action"`
	IsSystem   bool      `json:"isSystem"`
}

// UserRole is the user assignment relation (UA).
type UserRole struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy uint      `json:"createdBy"`
	UserID    uint      `json:"userId"`
	RoleID    uint      `json:"roleId"`
	IsSystem  bool      `json:"isSystem"`
}

// RoleHierarchy represents senior -> junior. The senior role inherits every
// permission authorized for the junior role. Multiple parents are allowed.
type RoleHierarchy struct {
	ID           uint      `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	CreatedBy    uint      `json:"createdBy"`
	SeniorRoleID uint      `json:"seniorRoleId"`
	JuniorRoleID uint      `json:"juniorRoleId"`
}

// SeparationConstraint defines an SSD or DSD role set. Cardinality is the
// smallest number of roles from the set that constitutes a violation.
type SeparationConstraint struct {
	ID          uint           `json:"id"`
	CreatedAt   time.Time      `json:"createdAt"`
	CreatedBy   uint           `json:"createdBy"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	UpdatedBy   uint           `json:"updatedBy"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        ConstraintType `json:"type"`
	Cardinality int            `json:"cardinality"`
	RoleIDs     []uint         `json:"roleIds"`
	IsEnabled   bool           `json:"isEnabled"`
}

// AuthorizationSession is the Core RBAC session relation. A session belongs
// to exactly one user and activates a subset of that user's authorized roles.
//
// Username, IpAddr, UserAgent and Remember describe the login that opened the
// session (the username is immutable, so the snapshot never goes stale);
// LastActiveAt advances whenever the session's token is refreshed.
type AuthorizationSession struct {
	ID           string     `json:"id"`
	UserID       uint       `json:"userId"`
	Username     string     `json:"username"`
	IpAddr       string     `json:"ipAddr"`
	UserAgent    string     `json:"userAgent"`
	Remember     bool       `json:"remember"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastActiveAt time.Time  `json:"lastActiveAt"`
	ExpiresAt    time.Time  `json:"expiresAt"`
	RevokedAt    *time.Time `json:"revokedAt,omitempty"`
	Version      uint64     `json:"version"`
}

type SessionRole struct {
	SessionID string `json:"sessionId"`
	RoleID    uint   `json:"roleId"`
}

type PermissionGrant struct {
	ResourceID uint     `json:"resourceId"`
	Actions    []string `json:"actions"`
}

type EffectivePermission struct {
	RoleID       uint   `json:"roleId"`
	RoleCode     string `json:"roleCode"`
	ResourceID   uint   `json:"resourceId"`
	ResourceCode string `json:"resourceCode"`
	ResourcePath string `json:"resourcePath"`
	Action       string `json:"action"`
}
