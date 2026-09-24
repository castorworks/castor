package shared

import "errors"

// ErrNotFound is returned by repositories when the requested record does not exist.
// Infrastructure implementations translate their storage-specific not-found errors to it,
// so the application layer never depends on an ORM.
var ErrNotFound = errors.New("record not found")

// ErrDuplicate is returned by repositories when a write violates a uniqueness rule.
var ErrDuplicate = errors.New("duplicate record")
