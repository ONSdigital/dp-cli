package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// InvokeLambda invokes a Lambda function with the provided JSON payload and returns the raw response payload as string.
func InvokeLambda(ctx context.Context, profile string, functionName string, payload []byte) (string, error) {
	cfg, err := GetAWSConfig(ctx, profile)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := lambda.NewFromConfig(cfg)

	out, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      payload,
	})
	if err != nil {
		return "", err
	}

	return string(out.Payload), nil
}
