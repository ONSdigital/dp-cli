package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func getAWSConfig(ctx context.Context, profile string) aws.Config {
	var configOpts []func(*config.LoadOptions) error

	configOpts = append(configOpts, config.WithRegion("eu-west-2"))

	if profile != "" {
		configOpts = append(configOpts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		configOpts...,
	)
	if err != nil {
		panic(err)
	}

	return cfg
}
