package mutate

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
)

type Signature map[string]string

func getSecurityContext(config *service.PiggyConfig, podSecurityContext *corev1.PodSecurityContext) *corev1.SecurityContext {
	sc := &corev1.SecurityContext{
		AllowPrivilegeEscalation: &config.PiggyPspAllowPrivilegeEscalation,
	}
	if podSecurityContext != nil && podSecurityContext.RunAsUser != nil {
		sc.RunAsUser = podSecurityContext.RunAsUser
	}
	return sc
}

func (m *Mutating) mutateCommand(config *service.PiggyConfig, container *corev1.Container, pod *corev1.Pod) ([]string, error) {
	// check if already mutated
	if len(container.Command) == 1 && container.Command[0] == "/piggy/piggy-env" {
		if len(container.Args) > 0 && container.Args[0] == "--" {
			// return original command (skipping the --)
			return container.Args[1:], nil
		}
		return container.Args, nil
	}
	entry := container.Command
	// if the container has no explicitly specified command
	if len(entry) == 0 {
		// read docker image
		imageConfig, err := m.registry.GetImageConfig(m.context, config, pod.Namespace, *container, pod.Spec)
		if err != nil {
			return nil, err
		}
		entry = append(entry, imageConfig.Entrypoint...)
		// If no Args are defined we can use the Docker CMD from the image
		// https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#notes
		if len(container.Args) == 0 {
			entry = append(entry, imageConfig.Cmd...)
		}
	}
	// append containers arguments
	entry = append(entry, container.Args...)
	// prepend piggy-env
	// insert --
	args := make([]string, len(entry)+1)
	args[0] = "--"
	copy(args[1:], entry)
	container.Command = []string{"/piggy/piggy-env"}
	container.Args = args
	return entry, nil
}

func (m *Mutating) mutateContainer(uid string, config *service.PiggyConfig, container *corev1.Container, pod *corev1.Pod) (string, error) {
	mutate := false
	var envVars []corev1.EnvVar
	if len(container.EnvFrom) > 0 {
		envFrom, err := m.LookForEnvFrom(container.EnvFrom, pod.Namespace)
		if err != nil {
			return "", fmt.Errorf("unable to read envFrom: %v", err)
		}
		envVars = append(envVars, envFrom...)
	}
	for _, env := range container.Env {
		if env.ValueFrom != nil {
			valueFrom, err := m.LookForValueFrom(env, pod.Namespace)
			if err != nil {
				return "", fmt.Errorf("unable to read valueFrom: %v", err)
			}
			if valueFrom != nil {
				envVars = append(envVars, *valueFrom)
			}
		} else {
			envVars = append(envVars, env)
		}
	}
	for _, env := range envVars {
		if strings.HasPrefix(env.Value, service.PrefixPiggy) {
			mutate = true
			break
		}
	}
	if !mutate {
		log.Debug().Str("namespace", pod.Namespace).Msgf("Skip mutating '%s' container ...", container.Name)
		return "", nil
	}
	// env vars to inject
	envs := []corev1.EnvVar{
		{
			Name:  "PIGGY_AWS_SECRET_NAME",
			Value: config.AWSSecretName,
		},
		{
			Name:  "PIGGY_AWS_REGION",
			Value: config.AWSRegion,
		},
		{
			Name:  "PIGGY_AWS_SSM_PARAMETER_PATH",
			Value: config.AWSSSMParameterPath,
		},
	}
	if config.Debug {
		envs = append(envs, corev1.EnvVar{Name: "PIGGY_DEBUG", Value: "true"})
	}
	if config.Standalone {
		envs = append(envs, corev1.EnvVar{Name: "PIGGY_STANDALONE", Value: "true"})
	} else if config.PiggyAddress != "" {
		envs = append(envs, corev1.EnvVar{
			Name:  "PIGGY_ADDRESS",
			Value: config.PiggyAddress,
		}, corev1.EnvVar{
			Name: "PIGGY_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		}, corev1.EnvVar{
			Name:  "PIGGY_UID",
			Value: uid,
		})
	}
	if config.PiggyIgnoreNoEnv {
		envs = append(envs, corev1.EnvVar{Name: "PIGGY_IGNORE_NO_ENV", Value: "true"})
	}
	if config.PiggyDNSResolver != "" {
		envs = append(envs, corev1.EnvVar{Name: "PIGGY_DNS_RESOLVER", Value: config.PiggyDNSResolver})
	}
	if config.PiggyInitialDelay != "" {
		if _, err := time.ParseDuration(config.PiggyInitialDelay); err == nil {
			envs = append(envs, corev1.EnvVar{Name: "PIGGY_INITIAL_DELAY", Value: config.PiggyInitialDelay})
		}
	}
	if config.PiggyNumberOfRetry > 0 {
		val := strconv.FormatInt(int64(config.PiggyNumberOfRetry), 10)
		envs = append(envs, corev1.EnvVar{Name: "PIGGY_NUMBER_OF_RETRY", Value: val})
	}

	for _, env := range envs {
		found := false
		for i, cEnv := range container.Env {
			if cEnv.Name == env.Name {
				container.Env[i] = env
				found = true
				break
			}
		}
		if !found {
			container.Env = append(container.Env, env)
		}
	}

	log.Debug().Str("namespace", pod.Namespace).Msgf("Modifying volume mounts '%s' containers ...", container.Name)
	foundVolumeMount := false
	for _, vm := range container.VolumeMounts {
		if vm.Name == service.VolumeNamePiggy {
			foundVolumeMount = true
			break
		}
	}
	if !foundVolumeMount {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      service.VolumeNamePiggy,
			MountPath: "/piggy/",
		})
	}
	log.Debug().Str("namespace", pod.Namespace).Msgf("Modifying command '%s' containers ...", container.Name)
	var args []string
	var err error
	if args, err = m.mutateCommand(config, container, pod); err != nil {
		log.Info().Str("namespace", pod.Namespace).Str("pod_name", pod.Name).Msgf("Error while mutating '%s' container command [%v]", container.Name, err)
	}
	// signature
	sig := strings.TrimSpace(strings.Join(args, " "))
	h := sha256.New()
	_, err = h.Write([]byte(sig))
	if err != nil {
		log.Error().Msgf("%v", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// MutatePod mutate pod
func (m *Mutating) MutatePod(config *service.PiggyConfig, pod *corev1.Pod) (interface{}, error) {
	start := time.Now()

	// Check if already mutated - Remove global short-circuit to support granular idempotency for reinvocations

	// Mutate pod only when it containing piggysec.com/aws-secret-name or piggysec.com/aws-ssm-parameter-path or piggysec.com/piggy-address annotation
	if config.AWSSecretName != "" || config.AWSSSMParameterPath != "" || config.PiggyAddress != "" {
		signature := make(Signature)
		log.Debug().Str("namespace", pod.Namespace).Msgf("Adding volumes to podspec ...")
		foundVolume := false
		for _, v := range pod.Spec.Volumes {
			if v.Name == service.VolumeNamePiggy {
				foundVolume = true
				break
			}
		}
		if !foundVolume {
			pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
				Name: service.VolumeNamePiggy,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						Medium: corev1.StorageMediumMemory,
					},
				},
			})
		}
		log.Debug().Str("namespace", pod.Namespace).Msgf("Mutating init-containers ...")
		for i := range pod.Spec.InitContainers {
			var err error
			uid := m.generateUID()
			sig, err := m.mutateContainer(uid, config, &pod.Spec.InitContainers[i], pod)
			if err != nil {
				return nil, err
			}
			if sig != "" {
				signature[uid] = sig
			}
		}
		log.Debug().Str("namespace", pod.Namespace).Msgf("Inserting init-container to podspec ...")
		foundInitContainer := false
		for _, c := range pod.Spec.InitContainers {
			if c.Name == "install-piggy-env" {
				foundInitContainer = true
				break
			}
		}
		if !foundInitContainer {
			initContainers := make([]corev1.Container, len(pod.Spec.InitContainers)+1)
			copy(initContainers[1:], pod.Spec.InitContainers)
			initContainers[0] = corev1.Container{
				Name:            "install-piggy-env",
				Image:           config.PiggyImage,
				ImagePullPolicy: config.PiggyImagePullPolicy,
				Args:            []string{"install", "/piggy"},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      service.VolumeNamePiggy,
						MountPath: "/piggy/",
					},
				},
				SecurityContext: getSecurityContext(config, pod.Spec.SecurityContext),
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    config.PiggyResourceCPULimit,
						corev1.ResourceMemory: config.PiggyResourceMemoryLimit,
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    config.PiggyResourceCPURequest,
						corev1.ResourceMemory: config.PiggyResourceMemoryRequest,
					},
				},
			}
			pod.Spec.InitContainers = initContainers
		}
		log.Debug().Str("namespace", pod.Namespace).Msgf("Mutating containers ...")
		for i := range pod.Spec.Containers {
			var err error
			uid := m.generateUID()
			sig, err := m.mutateContainer(uid, config, &pod.Spec.Containers[i], pod)
			if err != nil {
				return nil, err
			}
			if sig != "" {
				signature[uid] = sig
			}
		}
		bytes, err := json.Marshal(&signature)
		if err != nil {
			return nil, fmt.Errorf("marshaling signature: %v", err)
		}
		pod.Annotations[service.Namespace+service.ConfigPiggyUID] = string(bytes)
		// log
		logEvent := log.Info().Str("namespace", pod.Namespace)
		if pod.Name == "" && len(pod.OwnerReferences) > 0 {
			logEvent.Str("owner", pod.OwnerReferences[0].Name).Msgf("Pod of %s '%s' has been mutated (took %s)", pod.OwnerReferences[0].Kind, pod.OwnerReferences[0].Name, time.Since(start))
		} else {
			logEvent.Str("pod_name", pod.Name).Msgf("Pod '%s' has been mutated (took %s)", pod.Name, time.Since(start))
		}
		return pod, nil
	}
	log.Debug().Str("namespace", pod.Namespace).Str("pod_name", pod.Name).Msg("Skip mutating pod: No piggy annotations found.")
	return nil, nil
}
