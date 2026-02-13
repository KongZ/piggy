package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAWSClientFactory(t *testing.T) {
	f := &DefaultAWSClientFactory{}
	ctx := context.Background()
	region := "us-east-1"

	// Test SecretsManager
	sm, err := f.GetSecretsManagerClient(ctx, region)
	assert.NoError(t, err)
	assert.NotNil(t, sm)

	// Test SSM
	ssm, err := f.GetSSMClient(ctx, region)
	assert.NoError(t, err)
	assert.NotNil(t, ssm)
}
