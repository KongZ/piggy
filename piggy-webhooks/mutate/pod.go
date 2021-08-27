package mutate

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	if podSecurityContext.RunAsUser != nil {
		sc.RunAsUser = podSecurityContext.RunAsUser
	}
	return sc
}

func (m *Mutating) mutateCommand(config *service.PiggyConfig, container *corev1.Container, pod *corev1.Pod) ([]string, error) {
	entry := container.Command
	// if the container has no explicitly specified command
	if len(entry) == 0 {
		// read docker image
		imageConfig, err := m.registry.GetImageConfig(m.context, config, m.k8sClient, pod.ObjectMeta.Namespace, *container, pod.Spec)
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

// MutatePod mutate pod
func (m *Mutating) MutatePod(config *service.PiggyConfig, pod *corev1.Pod) (interface{}, error) {
	start := time.Now()
	// Mutate pod only when it containing piggysec.com/aws-secret-name annotation
	if config.AWSSecretName != "" || config.PiggyAddress != "" {
		uid := m.generateUid()
		signature := make(Signature)
		log.Debug().Str("namespace", pod.ObjectMeta.Namespace).Msgf("Adding volumes to podspec ...")
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: "piggy-env",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		})
		log.Debug().Str("namespace", pod.ObjectMeta.Namespace).Msgf("Adding init-containers to podspec ...")
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:            "install-piggy-env",
			Image:           config.PiggyImage,
			ImagePullPolicy: config.PiggyImagePullPolicy,
			Args:            []string{"install", "/piggy"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "piggy-env",
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
		})
		log.Debug().Str("namespace", pod.ObjectMeta.Namespace).Msgf("Mutating containers ...")
		for i := range pod.Spec.Containers {
			mutate := false
			for _, env := range pod.Spec.Containers[i].Env {
				if strings.HasPrefix(env.Value, "piggy:") {
					mutate = true
					break
				}
				if env.ValueFrom != nil {
					valueFrom, err := m.LookForValueFrom(env, pod.ObjectMeta.Namespace)
					if err != nil {
						return nil, fmt.Errorf("unable to read valueFrom: %v", err)
					}
					if valueFrom != nil {
						mutate = true
						break
					}
				}
			}
			if !mutate {
				log.Debug().Str("namespace", pod.ObjectMeta.Namespace).Msgf("Skip mutating '%s' container ...", pod.Spec.Containers[i].Name)
				continue
			}
			log.Debug().Str("namespace", pod.ObjectMeta.Namespace).Msgf("Modifying env '%s' container ...", pod.Spec.Containers[i].Name)
			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, []corev1.EnvVar{
				{
					Name:  "PIGGY_AWS_SECRET_NAME",
					Value: config.AWSSecretName,
				},
				{
					Name:  "PIGGY_AWS_REGION",
					Value: config.AWSRegion,
				},
			}...)
			if config.Debug {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, []corev1.EnvVar{
					{
						Name:  "PIGGY_DEBUG",
						Value: "true",
					},
				}...)
			}
			if config.Standalone {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, []corev1.EnvVar{
					{
						Name:  "PIGGY_STANDALONE",
						Value: "true",
					},
				}...)
			} else if config.PiggyAddress != "" {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, []corev1.EnvVar{
					{
						Name:  "PIGGY_ADDRESS",
						Value: config.PiggyAddress,
					},
					{
						Name: "PIGGY_POD_NAME",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.name",
							},
						},
					},
					{
						Name:  "PIGGY_UID",
						Value: uid,
					},
				}...)
			}
			if config.PiggyIgnoreNoEnv {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, []corev1.EnvVar{
					{
						Name:  "PIGGY_IGNORE_NO_ENV",
						Value: "true",
					},
				}...)
			}
			log.Debug().Str("namespace", pod.ObjectMeta.Namespace).Msgf("Modifying volume mounts '%s' containers ...", pod.Spec.Containers[i].Name)
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      "piggy-env",
				MountPath: "/piggy/",
			})
			log.Debug().Str("namespace", pod.ObjectMeta.Namespace).Msgf("Modifying command '%s' containers ...", pod.Spec.Containers[i].Name)
			var args []string
			var err error
			if args, err = m.mutateCommand(config, &pod.Spec.Containers[i], pod); err != nil {
				log.Info().Str("namespace", pod.ObjectMeta.Namespace).Str("pod_name", pod.ObjectMeta.Name).Msgf("Error while mutating '%s' container command [%v]", pod.Spec.Containers[i].Name, err)
			}
			// signature
			sig := strings.TrimSpace(strings.Join(args, " "))
			h := sha256.New()
			_, err = h.Write([]byte(sig))
			if err != nil {
				log.Error().Msgf("%v", err)
			}
			signature[uid] = fmt.Sprintf("%x", h.Sum(nil))
		}
		bytes, err := json.Marshal(&signature)
		if err != nil {
			return nil, fmt.Errorf("marshaling signature: %v", err)
		}
		pod.ObjectMeta.Annotations[service.Namespace+service.ConfigPiggyUID] = string(bytes)
		// log
		logEvent := log.Info().Str("namespace", pod.ObjectMeta.Namespace)
		if pod.ObjectMeta.Name == "" && len(pod.OwnerReferences) > 0 {
			logEvent.Str("owner", pod.OwnerReferences[0].Name).Msgf("Pod of %s '%s' has been mutated (took %s)", pod.OwnerReferences[0].Kind, pod.OwnerReferences[0].Name, time.Since(start))
		} else {
			logEvent.Str("pod_name", pod.ObjectMeta.Name).Msgf("Pod '%s' has been mutated (took %s)", pod.ObjectMeta.Name, time.Since(start))
		}
		return pod, nil
	}
	// for k, v := range pod.Annotations {
	// 	log.Info().Msgf("%s=%s", k, v)
	// }
	return nil, nil
}
