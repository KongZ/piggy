package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeEnv_Append(t *testing.T) {
	env := &sanitizedEnv{Env: []string{}}
	
	// Should append normal env
	env.append("MY_VAR", "my-value")
	assert.Contains(t, env.Env, "MY_VAR=my-value")
	
	// Should NOT append piggy env
	env.append("PIGGY_AWS_REGION", "us-east-1")
	assert.NotContains(t, env.Env, "PIGGY_AWS_REGION=us-east-1")
}

func TestDoSanitize(t *testing.T) {
	references := map[string]string{
		"DB_PASS": "piggy:db-pass",
		"API_KEY": "piggy:api-key",
		"NORMAL":  "value",
	}
	env := &sanitizedEnv{Env: []string{}}
	secrets := map[string]string{
		"db-pass": "secret123",
		"api-key": "key456",
	}
	
	doSanitize(references, env, secrets)
	
	assert.Contains(t, env.Env, "DB_PASS=secret123")
	assert.Contains(t, env.Env, "API_KEY=key456")
	assert.Contains(t, env.Env, "NORMAL=value")
}

func TestAwsErr(t *testing.T) {
	assert.False(t, awsErr(nil))
	// We can't easily mock smithy.APIError without more imports, but nil case is fine
}
