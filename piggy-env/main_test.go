package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSanitizeEnv_Append verifies that normal environment variables are appended
// while Piggy-specific ones are filtered out.
func TestSanitizeEnv_Append(t *testing.T) {
	env := &sanitizedEnv{Env: []string{}}

	// Should append normal env
	env.append("MY_VAR", "my-value")
	assert.Contains(t, env.Env, "MY_VAR=my-value")

	// Should NOT append piggy env
	env.append("PIGGY_AWS_REGION", "us-east-1")
	assert.NotContains(t, env.Env, "PIGGY_AWS_REGION=us-east-1")
}

// TestDoSanitize checks the core logic of replacing Piggy secret references with actual values.
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

// TestAwsErr ensures that a nil error returns false for being an AWS API error.
func TestAwsErr(t *testing.T) {
	assert.False(t, awsErr(nil))
	// We can't easily mock smithy.APIError without more imports, but nil case is fine
}
