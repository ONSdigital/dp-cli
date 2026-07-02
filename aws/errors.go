package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sts"
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
		strings.Contains(msg, "UnauthorizedAccess") ||
		strings.Contains(msg, "failed to refresh cached credentials")
}

// ValidateCredentials checks that the given profile has valid, active credentials
// by calling sts:GetCallerIdentity. Returns nil if credentials are valid.
func ValidateCredentials(ctx context.Context, profile string) error {
	cfg, err := GetAWSConfig(ctx, profile)
	if err != nil {
		return err
	}
	client := sts.NewFromConfig(cfg)
	_, err = client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	return err
}
