package mutate

import (
	"context"
	"fmt"

	"github.com/KongZ/piggy/piggy-webhooks/service"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
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

func (m *Mutating) mergeConfig(config *service.PiggyConfig, annotations map[string]string) *service.PiggyConfig {
	config.PiggyImage = service.GetStringValue(annotations, service.ConfigPiggyEnvImage, "")
	config.PiggyImagePullPolicy = corev1.PullPolicy(service.GetStringValue(annotations, service.ConfigPiggyEnvImagePullPolicy, "Always"))
	config.PiggyResourceCPURequest, _ = resource.ParseQuantity(service.GetStringValue(annotations, service.ConfigPiggyEnvResourceCPURequest, "50m"))
	config.PiggyResourceMemoryRequest, _ = resource.ParseQuantity(service.GetStringValue(annotations, service.ConfigPiggyEnvResourceMemoryRequest, "64Mi"))
	config.PiggyResourceCPULimit, _ = resource.ParseQuantity(service.GetStringValue(annotations, service.ConfigPiggyEnvResourceCPULimit, "200m"))
	config.PiggyResourceMemoryLimit, _ = resource.ParseQuantity(service.GetStringValue(annotations, service.ConfigPiggyEnvResourceMemoryLimit, "64Mi"))
	config.PiggyPspAllowPrivilegeEscalation = service.GetBoolValue(annotations, service.ConfigPiggyPSPAllowPrivilegeEscalation, false)
	config.PiggyAddress = service.GetStringValue(annotations, service.ConfigPiggyAddress, "")
	config.AWSSecretName = service.GetStringValue(annotations, service.AWSSecretName, "")
	config.AWSRegion = service.GetStringValue(annotations, service.ConfigAWSRegion, "")
	config.Debug = service.GetBoolValue(annotations, service.ConfigDebug, false)
	config.ImagePullSecret = service.GetStringValue(annotations, service.ConfigImagePullSecret, "")
	config.ImagePullSecretNamespace = service.GetStringValue(annotations, service.ConfigImagePullSecretNamespace, "")
	config.ImageSkipVerifyRegistry = service.GetBoolValue(annotations, service.ConfigImageSkipVerifyRegistry, true)
	config.Standalone = service.GetBoolValue(annotations, service.ConfigStandalone, false)
	return config
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
		config = m.mergeConfig(config, pod.Annotations)
		m.registry = NewRegistry(config)
		return m.MutatePod(config, &pod)
	}
	return nil, nil
}
