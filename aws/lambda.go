package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

func getLambdaClient(ctx context.Context, profile string) *lambda.Client {
	return lambda.NewFromConfig(getAWSConfig(ctx, profile))
}

// InvokeLambda invokes a Lambda function with the provided JSON payload and returns the raw response payload as string.
func InvokeLambda(ctx context.Context, profile string, functionName string, payload []byte) (string, error) {
	client := getLambdaClient(ctx, profile)

	out, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      payload,
	})
	if err != nil {
		return "", err
	}

	return string(out.Payload), nil
}
