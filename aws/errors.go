package aws

import (
	"strings"
)

// IsAccessError checks if an error is an access/permission issue.
func IsAccessError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "ForbiddenException") ||
		strings.Contains(msg, "No access") ||
		strings.Contains(msg, "not authorized") ||
		strings.Contains(msg, "AccessDenied") ||
		strings.Contains(msg, "UnauthorizedAccess")
}
