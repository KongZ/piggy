package service

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type MockSecretsManagerClient struct {
	GetSecretValueFunc func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func (m *MockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.GetSecretValueFunc != nil {
		return m.GetSecretValueFunc(ctx, params, optFns...)
	}
	return &secretsmanager.GetSecretValueOutput{}, nil
}

type MockSSMClient struct {
	GetParametersByPathFunc func(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

func (m *MockSSMClient) GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if m.GetParametersByPathFunc != nil {
		return m.GetParametersByPathFunc(ctx, params, optFns...)
	}
	return &ssm.GetParametersByPathOutput{}, nil
}

type MockAWSClientFactory struct {
	GetSecretsManagerClientFunc func(ctx context.Context, region string) (SecretsManagerClient, error)
	GetSSMClientFunc            func(ctx context.Context, region string) (SSMClient, error)
}

func (m *MockAWSClientFactory) GetSecretsManagerClient(ctx context.Context, region string) (SecretsManagerClient, error) {
	if m.GetSecretsManagerClientFunc != nil {
		return m.GetSecretsManagerClientFunc(ctx, region)
	}
	return &MockSecretsManagerClient{}, nil
}

func (m *MockAWSClientFactory) GetSSMClient(ctx context.Context, region string) (SSMClient, error) {
	if m.GetSSMClientFunc != nil {
		return m.GetSSMClientFunc(ctx, region)
	}
	return &MockSSMClient{}, nil
}
