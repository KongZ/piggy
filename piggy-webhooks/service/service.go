package service

import (
	"os"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const Namespace = "piggy.kong-z.com/"
const AWSSecretName = "aws-secret-name"                                         // AWS secret name
const ConfigAWSRegion = "aws-region"                                            // AWS secret's region
const ConfigPiggyEnvImage = "piggy-env-image"                                   // The piggy-env image URL
const ConfigPiggyEnvImagePullPolicy = "piggy-env-image-pull-policy"             // The piggy-env image pull policy
const ConfigPiggyEnvResourceCPURequest = "piggy-env-resource-cpu-request"       // The piggy-env init-container cpu request
const ConfigPiggyEnvResourceMemoryRequest = "piggy-env-resource-memory-request" // The piggy-env init-container memory request
const ConfigPiggyEnvResourceCPULimit = "piggy-env-resource-cpu-limit"           // The piggy-env init-container cpu limit
const ConfigPiggyEnvResourceMemoryLimit = "piggy-env-resource-memory-limit"     // The piggy-env init-container memory request
const ConfigPiggyPSPAllowPrivilegeEscalation = "psp-allow-privilege-escalation" // Default to false; not allow init-container to run as root
const ConfigPiggyAddress = "piggy-address"                                      // The endpoint of piggy-webhook
const ConfigPiggySkipVerifyTLS = "piggy-skip-verify-tls"                        // Default to true; Allow to skip verify TLS connection at piggy-address
const ConfigPiggyUID = "piggy-uid"                                              // A piggy uid
const ConfigDebug = "debug"                                                     // Enable debuging log
const ConfigImagePullSecret = "image-pull-secret"                               // Container image pull secret
const ConfigImagePullSecretNamespace = "image-pull-secret-namespace"            // Container image pull secret namespace
const ConfigImageSkipVerifyRegistry = "image-skip-verify-registry"              // Default to true; not verify the registry
const ConfigStandalone = "standalone"                                           // Default to false; use piggy-webhook to read secrets instead of pod

type PiggyConfig struct {
	PiggyImage                       string            `json:"piggyImage"`
	PiggyImagePullPolicy             corev1.PullPolicy `json:"piggyImagePullPolicy"`
	PiggyResourceCPURequest          resource.Quantity `json:"piggyResourceCPURequest"`
	PiggyResourceMemoryRequest       resource.Quantity `json:"piggyResourceMemoryRequest"`
	PiggyResourceCPULimit            resource.Quantity `json:"piggyResourceCPULimit"`
	PiggyResourceMemoryLimit         resource.Quantity `json:"piggyResourceMemoryLimit"`
	PiggyPspAllowPrivilegeEscalation bool              `json:"piggyPspAllowPrivilegeEscalation"`
	PiggyAddress                     string            `json:"piggyAddress"`
	PiggySkipVerifyTLS               string            `json:"piggySkipVerifyTLS"`
	PiggyUID                         string            `json:"piggyUID"`
	AWSSecretName                    string            `json:"awsSecretName"`
	AWSRegion                        string            `json:"awsRegion"`
	Debug                            bool              `json:"debug"`
	ImagePullSecret                  string            `json:"imagePullSecret"`
	ImagePullSecretNamespace         string            `json:"imagePullSecretNamespace"`
	ImageSkipVerifyRegistry          bool              `json:"imageSkipVerifyRegistry"`
	Standalone                       bool              `json:"standalone"`
	//
	PodServiceAccountName string
}

// GetEnv get environment value or return default value if not found
func GetEnv(name string, defaultValue string) string {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetEnvBool get environment value as bool or return default value if not found
func GetEnvBool(name string, defaultValue bool) bool {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	b, _ := strconv.ParseBool(val)
	return b
}

// GetStringValue get a string value from annotation map
func GetStringValue(annotations map[string]string, name string, defaultValue string) string {
	if val, ok := annotations[Namespace+name]; ok {
		return val
	}
	envName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return GetEnv(envName, defaultValue)
}

// GetBoolValue get a boolean value from annotation map
func GetBoolValue(annotations map[string]string, name string, defaultValue bool) bool {
	if val, ok := annotations[Namespace+name]; ok {
		b, _ := strconv.ParseBool(val)
		return b
	}
	envName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return GetEnvBool(envName, defaultValue)
}
