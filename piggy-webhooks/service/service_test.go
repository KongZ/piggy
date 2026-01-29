package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetEnv checks the retrieval of environment variables with defaults.
func TestGetEnv(t *testing.T) {
	_ = os.Setenv("TEST_KEY", "test_value")
	defer func() { _ = os.Unsetenv("TEST_KEY") }()

	assert.Equal(t, "test_value", GetEnv("TEST_KEY", "default"))
	assert.Equal(t, "default", GetEnv("NON_EXISTENT", "default"))
}

// TestGetEnvBool checks the boolean parsing of environment variables.
func TestGetEnvBool(t *testing.T) {
	_ = os.Setenv("TEST_BOOL_TRUE", "true")
	_ = os.Setenv("TEST_BOOL_FALSE", "false")
	defer func() {
		_ = os.Unsetenv("TEST_BOOL_TRUE")
		_ = os.Unsetenv("TEST_BOOL_FALSE")
	}()

	assert.True(t, GetEnvBool("TEST_BOOL_TRUE", false))
	assert.False(t, GetEnvBool("TEST_BOOL_FALSE", true))
	assert.True(t, GetEnvBool("NON_EXISTENT", true))
	assert.False(t, GetEnvBool("NON_EXISTENT", false))
}

// TestGetEnvInt checks the integer parsing of environment variables.
func TestGetEnvInt(t *testing.T) {
	_ = os.Setenv("TEST_INT", "123")
	_ = os.Setenv("TEST_INVALID_INT", "abc")
	defer func() {
		_ = os.Unsetenv("TEST_INT")
		_ = os.Unsetenv("TEST_INVALID_INT")
	}()

	assert.Equal(t, 123, GetEnvInt("TEST_INT", 0))
	assert.Equal(t, 0, GetEnvInt("TEST_INVALID_INT", 0))
	assert.Equal(t, 456, GetEnvInt("NON_EXISTENT", 456))
}

// TestGetStringValue verifies string retrieval from both annotations and environment variables.
func TestGetStringValue(t *testing.T) {
	annotations := map[string]string{
		Namespace + "test-string": "annotation_value",
	}
	_ = os.Setenv("TEST_STRING", "env_value")
	defer func() { _ = os.Unsetenv("TEST_STRING") }()

	assert.Equal(t, "annotation_value", GetStringValue(annotations, "test-string", "default"))
	assert.Equal(t, "env_value", GetStringValue(nil, "test-string", "default"))
	assert.Equal(t, "default", GetStringValue(nil, "non-existent", "default"))
}

// TestGetBoolValue verifies boolean retrieval from both annotations and environment variables.
func TestGetBoolValue(t *testing.T) {
	annotations := map[string]string{
		Namespace + "test-bool": "true",
	}
	_ = os.Setenv("TEST_BOOL", "false")
	defer func() { _ = os.Unsetenv("TEST_BOOL") }()

	assert.True(t, GetBoolValue(annotations, "test-bool", false))
	assert.False(t, GetBoolValue(nil, "test-bool", true))
	assert.True(t, GetBoolValue(nil, "non-existent", true))
}

// TestGetIntValue verifies integer retrieval from both annotations and environment variables.
func TestGetIntValue(t *testing.T) {
	annotations := map[string]string{
		Namespace + "test-int": "123",
	}
	_ = os.Setenv("TEST_INT", "456")
	defer func() { _ = os.Unsetenv("TEST_INT") }()

	assert.Equal(t, 123, GetIntValue(annotations, "test-int", 0))
	assert.Equal(t, 456, GetIntValue(nil, "test-int", 0))
	assert.Equal(t, 789, GetIntValue(nil, "non-existent", 789))

	invalidAnnotations := map[string]string{
		Namespace + "test-int": "abc",
	}
	assert.Equal(t, 0, GetIntValue(invalidAnnotations, "test-int", 0))
}
