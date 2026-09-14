package query

import (
	"errors"
	"regexp"
	"strings"
)

// ErrInvalidOrder is returned when a sort expression is malformed or references a
// column outside the allowlist.
var ErrInvalidOrder = errors.New("invalid order")

// MaxOrderTerms bounds the number of comma-separated sort terms accepted from clients.
const MaxOrderTerms = 3

var orderColumnPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// NormalizeOrder validates a client supplied sort expression and rebuilds it from
// allowlisted tokens only, so the returned string is safe to hand to an ORM.
//
// Accepted grammar: term[, term]... where term is `column [asc|desc]`.
// Columns are compared case-insensitively against allowed; the direction defaults
// to ASC. An empty expression returns an empty string.
func NormalizeOrder(order string, allowed func(column string) bool) (string, error) {
	order = strings.TrimSpace(order)
	if order == "" {
		return "", nil
	}
	if allowed == nil {
		return "", ErrInvalidOrder
	}
	terms := strings.Split(order, ",")
	if len(terms) > MaxOrderTerms {
		return "", ErrInvalidOrder
	}
	normalized := make([]string, 0, len(terms))
	for _, term := range terms {
		parts := strings.Fields(term)
		if len(parts) == 0 || len(parts) > 2 {
			return "", ErrInvalidOrder
		}
		column := strings.ToLower(parts[0])
		if !orderColumnPattern.MatchString(column) || !allowed(column) {
			return "", ErrInvalidOrder
		}
		direction := "ASC"
		if len(parts) == 2 {
			switch strings.ToLower(parts[1]) {
			case "asc":
			case "desc":
				direction = "DESC"
			default:
				return "", ErrInvalidOrder
			}
		}
		normalized = append(normalized, column+" "+direction)
	}
	return strings.Join(normalized, ", "), nil
}

// AllowColumns returns an allowlist predicate backed by a fixed set of columns.
func AllowColumns(columns ...string) func(string) bool {
	set := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		set[strings.ToLower(column)] = struct{}{}
	}
	return func(column string) bool {
		_, ok := set[column]
		return ok
	}
}

// sensitiveColumnMarkers lists column name fragments that must never be used for
// sorting or filtering, even when the column exists on the model.
var sensitiveColumnMarkers = []string{"password", "secret", "token", "private_key", "salt"}

// IsSensitiveColumn reports whether a column holds credentials or other secrets.
func IsSensitiveColumn(column string) bool {
	column = strings.ToLower(column)
	for _, marker := range sensitiveColumnMarkers {
		if strings.Contains(column, marker) {
			return true
		}
	}
	return false
}
