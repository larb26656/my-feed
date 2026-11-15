package database

import "strings"

// IsUniqueConstraintViolation reports SQLite uniqueness constraint errors. If
// a constraint detail is provided, the error must also contain that detail.
func IsUniqueConstraintViolation(err error, constraintDetails ...string) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if !strings.Contains(message, "unique constraint failed") {
		return false
	}
	for _, detail := range constraintDetails {
		if !strings.Contains(message, strings.ToLower(detail)) {
			return false
		}
	}
	return true
}
