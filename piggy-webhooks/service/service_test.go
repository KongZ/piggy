package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	assert.Equal(t, "test_value", GetEnv("TEST_KEY", "default"))
	assert.Equal(t, "default", GetEnv("NON_EXISTENT", "default"))
}

func TestGetEnvBool(t *testing.T) {
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "false")
	defer os.Unsetenv("TEST_BOOL_TRUE")
	defer os.Unsetenv("TEST_BOOL_FALSE")

	assert.True(t, GetEnvBool("TEST_BOOL_TRUE", false))
	assert.False(t, GetEnvBool("TEST_BOOL_FALSE", true))
	assert.True(t, GetEnvBool("NON_EXISTENT", true))
	assert.False(t, GetEnvBool("NON_EXISTENT", false))
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "123")
	os.Setenv("TEST_INVALID_INT", "abc")
	defer os.Unsetenv("TEST_INT")
	defer os.Unsetenv("TEST_INVALID_INT")

	assert.Equal(t, 123, GetEnvInt("TEST_INT", 0))
	assert.Equal(t, 0, GetEnvInt("TEST_INVALID_INT", 0))
	assert.Equal(t, 456, GetEnvInt("NON_EXISTENT", 456))
}

func TestGetStringValue(t *testing.T) {
	annotations := map[string]string{
		Namespace + "test-string": "annotation_value",
	}
	os.Setenv("TEST_STRING", "env_value")
	defer os.Unsetenv("TEST_STRING")

	assert.Equal(t, "annotation_value", GetStringValue(annotations, "test-string", "default"))
	assert.Equal(t, "env_value", GetStringValue(nil, "test-string", "default"))
	assert.Equal(t, "default", GetStringValue(nil, "non-existent", "default"))
}

func TestGetBoolValue(t *testing.T) {
	annotations := map[string]string{
		Namespace + "test-bool": "true",
	}
	os.Setenv("TEST_BOOL", "false")
	defer os.Unsetenv("TEST_BOOL")

	assert.True(t, GetBoolValue(annotations, "test-bool", false))
	assert.False(t, GetBoolValue(nil, "test-bool", true))
	assert.True(t, GetBoolValue(nil, "non-existent", true))
}

func TestGetIntValue(t *testing.T) {
	annotations := map[string]string{
		Namespace + "test-int": "123",
	}
	os.Setenv("TEST_INT", "456")
	defer os.Unsetenv("TEST_INT")

	assert.Equal(t, 123, GetIntValue(annotations, "test-int", 0))
	assert.Equal(t, 456, GetIntValue(nil, "test-int", 0))
	assert.Equal(t, 789, GetIntValue(nil, "non-existent", 789))

	invalidAnnotations := map[string]string{
		Namespace + "test-int": "abc",
	}
	assert.Equal(t, 0, GetIntValue(invalidAnnotations, "test-int", 0))
}
