package mutate

import (
	"context"
	"fmt"
	"strings"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	"github.com/google/uuid"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
)

var (
	podResource           = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
	UniversalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
)

// Mutating a mutating object
type Mutating struct {
	registry  *ImageRegistry
	k8sClient kubernetes.Interface
	context   context.Context
}

// IsKubeNamespace checks if the given namespace is a Kubernetes-owned namespace.
func IsKubeNamespace(ns string) bool {
	return ns == metav1.NamespacePublic || ns == metav1.NamespaceSystem
}

// NewMutating create mutating object
func NewMutating(ctx context.Context, k8sClient kubernetes.Interface) (*Mutating, error) {
	mutating := &Mutating{
		context:   ctx,
		k8sClient: k8sClient,
	}
	return mutating, nil
}

// GenerateUid get an uid
func (m *Mutating) generateUid() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func (m *Mutating) mergeConfig(config *service.PiggyConfig, annotations map[string]string) *service.PiggyConfig {
	config.PiggyImage = service.GetStringValue(annotations, service.ConfigPiggyEnvImage, "")
	config.PiggyImagePullPolicy = corev1.PullPolicy(service.GetStringValue(annotations, service.ConfigPiggyEnvImagePullPolicy, "Always"))
	config.PiggyResourceCPURequest, _ = resource.ParseQuantity(service.GetStringValue(annotations, service.ConfigPiggyEnvResourceCPURequest, "50m"))
	config.PiggyResourceMemoryRequest, _ = resource.ParseQuantity(service.GetStringValue(annotations, service.ConfigPiggyEnvResourceMemoryRequest, "64Mi"))
	config.PiggyResourceCPULimit, _ = resource.ParseQuantity(service.GetStringValue(annotations, service.ConfigPiggyEnvResourceCPULimit, "200m"))
	config.PiggyResourceMemoryLimit, _ = resource.ParseQuantity(service.GetStringValue(annotations, service.ConfigPiggyEnvResourceMemoryLimit, "64Mi"))
	config.PiggyPspAllowPrivilegeEscalation = service.GetBoolValue(annotations, service.ConfigPiggyPSPAllowPrivilegeEscalation, false)
	config.PiggyAddress = service.GetStringValue(annotations, service.ConfigPiggyAddress, "")
	config.PiggyIgnoreNoEnv = service.GetBoolValue(annotations, service.ConfigPiggyIgnoreNoEnv, false)
	config.PiggyEnforceIntegrity = service.GetBoolValue(annotations, service.ConfigPiggyEnforceIntegrity, true)
	config.AWSSecretName = service.GetStringValue(annotations, service.AWSSecretName, "")
	config.AWSRegion = service.GetStringValue(annotations, service.ConfigAWSRegion, "")
	config.Debug = service.GetBoolValue(annotations, service.ConfigDebug, false)
	config.ImagePullSecret = service.GetStringValue(annotations, service.ConfigImagePullSecret, "")
	config.ImagePullSecretNamespace = service.GetStringValue(annotations, service.ConfigImagePullSecretNamespace, "")
	config.ImageSkipVerifyRegistry = service.GetBoolValue(annotations, service.ConfigImageSkipVerifyRegistry, true)
	config.Standalone = service.GetBoolValue(annotations, service.ConfigStandalone, false)
	config.PiggyDNSResolver = service.GetStringValue(annotations, service.ConfigPiggyDNSResolver, "")
	config.PiggyDelaySecond = service.GetIntValue(annotations, service.ConfigPiggyDelaySecond, 0)
	return config
}

func (m *Mutating) getDataFromConfigmap(configMapName string, ns string) (map[string]string, error) {
	configMap, err := m.k8sClient.CoreV1().ConfigMaps(ns).Get(context.Background(), configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configMap.Data, nil
}

func (m *Mutating) getDataFromSecret(secretName string, ns string) (map[string][]byte, error) {
	secret, err := m.k8sClient.CoreV1().Secrets(ns).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

// ApplyPiggy handle adminssion request and apply piggy to pod
func (m *Mutating) ApplyPiggy(req *admissionv1.AdmissionRequest) (interface{}, error) {
	config := &service.PiggyConfig{}
	if req.Resource == podResource {
		// Parse the Pod object.
		raw := req.Object.Raw
		pod := corev1.Pod{}
		if _, _, err := UniversalDeserializer.Decode(raw, nil, &pod); err != nil {
			return nil, fmt.Errorf("could not deserialize pod object: %v", err)
		}
		// for unknown reason, object meta is missing in some cluster
		if pod.ObjectMeta.Namespace == "" {
			pod.ObjectMeta.Namespace = req.Namespace
		}
		config = m.mergeConfig(config, pod.Annotations)
		m.registry = NewRegistry(config)
		return m.MutatePod(config, &pod)
	}
	return nil, nil
}

// LookForValueFrom look up value from valueFrom
func (m *Mutating) LookForValueFrom(env corev1.EnvVar, ns string) (*corev1.EnvVar, error) {
	if env.ValueFrom.ConfigMapKeyRef != nil {
		data, err := m.getDataFromConfigmap(env.ValueFrom.ConfigMapKeyRef.Name, ns)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		value := data[env.ValueFrom.ConfigMapKeyRef.Key]
		if strings.HasPrefix(value, "piggy:") {
			fromCM := corev1.EnvVar{
				Name:  env.Name,
				Value: value,
			}
			return &fromCM, nil
		}
	}
	if env.ValueFrom.SecretKeyRef != nil {
		data, err := m.getDataFromSecret(env.ValueFrom.SecretKeyRef.Name, ns)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		value := string(data[env.ValueFrom.SecretKeyRef.Key])
		if strings.HasPrefix(value, "piggy:") {
			fromSecret := corev1.EnvVar{
				Name:  env.Name,
				Value: value,
			}
			return &fromSecret, nil
		}
	}
	return nil, nil
}

// LookForEnvFrom look up value from envFrom
func (mw *Mutating) LookForEnvFrom(envFrom []corev1.EnvFromSource, ns string) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	for _, ef := range envFrom {
		if ef.ConfigMapRef != nil {
			data, err := mw.getDataFromConfigmap(ef.ConfigMapRef.Name, ns)
			if err != nil {
				if apierrors.IsNotFound(err) || (ef.ConfigMapRef.Optional != nil && *ef.ConfigMapRef.Optional) {
					continue
				} else {
					return envVars, err
				}
			}
			for key, value := range data {
				if strings.HasPrefix(value, "piggy:") {
					fromCM := corev1.EnvVar{
						Name:  key,
						Value: value,
					}
					envVars = append(envVars, fromCM)
				}
			}
		}
		if ef.SecretRef != nil {
			data, err := mw.getDataFromSecret(ef.SecretRef.Name, ns)
			if err != nil {
				if apierrors.IsNotFound(err) || (ef.SecretRef.Optional != nil && *ef.SecretRef.Optional) {
					continue
				} else {
					return envVars, err
				}
			}
			for key, v := range data {
				value := string(v)
				if strings.HasPrefix(value, "piggy:") {
					fromSecret := corev1.EnvVar{
						Name:  key,
						Value: value,
					}
					envVars = append(envVars, fromSecret)
				}
			}
		}
	}
	return envVars, nil
}
