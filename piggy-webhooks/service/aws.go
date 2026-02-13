package service

import (
	"context"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SecretsManagerClient defines the interface for AWS Secrets Manager client
type SecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// SSMClient defines the interface for AWS SSM client
type SSMClient interface {
	GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

// AWSClientFactory defines the interface for creating AWS clients
type AWSClientFactory interface {
	GetSecretsManagerClient(ctx context.Context, region string) (SecretsManagerClient, error)
	GetSSMClient(ctx context.Context, region string) (SSMClient, error)
}

// DefaultAWSClientFactory is the default implementation that creates real AWS clients
type DefaultAWSClientFactory struct{}

func (f *DefaultAWSClientFactory) GetSecretsManagerClient(ctx context.Context, region string) (SecretsManagerClient, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return secretsmanager.NewFromConfig(cfg), nil
}

func (f *DefaultAWSClientFactory) GetSSMClient(ctx context.Context, region string) (SSMClient, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return ssm.NewFromConfig(cfg), nil
}
