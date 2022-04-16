package service

import (
	"errors"
	"os"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	EmptyMap = make(map[string]string)
	// ErrorAuthorized when requestor does not have a permission
	ErrorAuthorized = errors.New("decision not allowed")
)

const Namespace = "piggysec.com/"
const AWSSecretName = "aws-secret-name"                                               // AWS secret name
const AWSSSMParameterPath = "aws-ssm-parameter-path"                                  // AWS SSM parameter path
const ConfigAWSRegion = "aws-region"                                                  // AWS secret's region
const ConfigPiggyEnvImage = "piggy-env-image"                                         // The piggy-env image URL
const ConfigPiggyEnvImagePullPolicy = "piggy-env-image-pull-policy"                   // The piggy-env image pull policy
const ConfigPiggyEnvResourceCPURequest = "piggy-env-resource-cpu-request"             // The piggy-env init-container cpu request
const ConfigPiggyEnvResourceMemoryRequest = "piggy-env-resource-memory-request"       // The piggy-env init-container memory request
const ConfigPiggyEnvResourceCPULimit = "piggy-env-resource-cpu-limit"                 // The piggy-env init-container cpu limit
const ConfigPiggyEnvResourceMemoryLimit = "piggy-env-resource-memory-limit"           // The piggy-env init-container memory request
const ConfigPiggyPSPAllowPrivilegeEscalation = "piggy-psp-allow-privilege-escalation" // Default to false; not allow init-container to run as root
const ConfigPiggyAddress = "piggy-address"                                            // The endpoint of piggy-webhook
const ConfigPiggySkipVerifyTLS = "piggy-skip-verify-tls"                              // Default to true; Allow to skip verify TLS connection at piggy-address
const ConfigPiggyUID = "piggy-uid"                                                    // A piggy uid
const ConfigPiggyIgnoreNoEnv = "piggy-ignore-no-env"                                  // Default to false; Exit piggy-env if no environment variable found on secret manager
const ConfigPiggyEnforceIntegrity = "piggy-enforce-integrity"                         // Default to true; Check the command integrity before run.
const ConfigDebug = "debug"                                                           // Enable debuging log
// #nosec G101 it is not a credential
const ConfigImagePullSecret = "image-pull-secret" // Container image pull secret
// #nosec G101 it is not a credential
const ConfigImagePullSecretNamespace = "image-pull-secret-namespace" // Container image pull secret namespace
const ConfigImageSkipVerifyRegistry = "image-skip-verify-registry"   // Default to true; not verify the registry
const ConfigStandalone = "standalone"                                // Default to false; use piggy-webhook to read secrets instead of pod
const ConfigPiggyDNSResolver = "piggy-dns-resolver"                  // Default to ""; Set Golang DNS resolver such as `tcp`, `udp`. See https://pkg.go.dev/net
const ConfigPiggyInitialDelay = "piggy-initial-delay"                // Default to 0; Delay n[ns|us|ms|s|m|h] before requesting secret from piggy-webhooks or secret-manager e.g. 1s (1 second)
const ConfigPiggyNumberOfRetry = "piggy-number-of-retry"             // Default to 0; Set number of retry retrieving secrets before giving up
// use only when injecting secrets
const ConfigPiggyEnforceServiceAccount = "piggy-enforce-service-account"      // Default to false; Force to check `PIGGY_ALLOWED_SA` env value in AWS secret manager
const ConfigPiggyDefaultSecretNamePrefix = "piggy-default-secret-name-prefix" // Default to ""; Set default prefix string for secret name
const ConfigPiggyDefaultSecretNameSuffix = "piggy-default-secret-name-suffix" // Default to ""; Set default suffix string for secret name

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
	PiggyIgnoreNoEnv                 bool              `json:"piggyIgnoreNoEnv"`
	PiggyEnforceIntegrity            bool              `json:"piggyEnforceIntegrity"`
	AWSSecretName                    string            `json:"awsSecretName"`
	AWSRegion                        string            `json:"awsRegion"`
	AWSSSMParameterPath              string            `json:"awsSSMParameterPath"`
	Debug                            bool              `json:"debug"`
	ImagePullSecret                  string            `json:"imagePullSecret"`
	ImagePullSecretNamespace         string            `json:"imagePullSecretNamespace"`
	ImageSkipVerifyRegistry          bool              `json:"imageSkipVerifyRegistry"`
	Standalone                       bool              `json:"standalone"`
	PiggyDNSResolver                 string            `json:"piggyDNSResolver"`
	PiggyInitialDelay                string            `json:"piggyInitialDelay"`
	PiggyNumberOfRetry               int               `json:"piggyNumberOfRetry"`
	// use only when injecting secrets
	PiggyEnforceServiceAccount   bool   `json:"piggyEnforceServiceAccount"`
	PiggyDefaultSecretNamePrefix string `json:"piggyDefaultSecretNamePrefix"`
	PiggyDefaultSecretNameSuffix string `json:"piggyDefaultSecretNameSuffix"`
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

// GetEnvInt get environment value as int or return default value if not found
func GetEnvInt(name string, defaultValue int) int {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseInt(val, 10, 0)
	if err != nil {
		return defaultValue
	}
	return int(b)
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

// GetIntValue get a int value from annotation map
func GetIntValue(annotations map[string]string, name string, defaultValue int) int {
	if val, ok := annotations[Namespace+name]; ok {
		b, err := strconv.ParseInt(val, 10, 0)
		if err != nil {
			return defaultValue
		}
		return int(b)
	}
	envName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return GetEnvInt(envName, defaultValue)
}
