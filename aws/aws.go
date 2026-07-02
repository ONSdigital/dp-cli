package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// GetAWSConfig loads the AWS SDK config for the given profile.
// Region is determined by the SDK's default resolution chain
// (environment variables, shared config file, then falls back to eu-west-2).
func GetAWSConfig(ctx context.Context, profile string) (aws.Config, error) {
	var configOpts []func(*config.LoadOptions) error

	configOpts = append(configOpts, config.WithRegion("eu-west-2"))

	if profile != "" {
		configOpts = append(configOpts, config.WithSharedConfigProfile(profile))
	}

	return config.LoadDefaultConfig(ctx, configOpts...)
}
