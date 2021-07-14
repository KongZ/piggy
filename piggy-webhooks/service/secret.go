package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/rs/zerolog/log"
)

type SanitizedEnv map[string]string

type GetSecretPayload struct {
	Namespace string `json:"namespace"`
	Resources string `json:"resources"`
	Name      string `json:"name"`
}

type Service struct {
	context   context.Context
	k8sClient kubernetes.Interface
}

// NewService new service
func NewService(ctx context.Context, k8sClient kubernetes.Interface) (*Service, error) {
	svc := &Service{
		context:   ctx,
		k8sClient: k8sClient,
	}
	return svc, nil
}

var sanitizeEnvmap = map[string]bool{
	"PIGGY_AWS_SECRET_NAME": true,
	"PIGGY_AWS_REGION":      true,
	"PIGGY_POD_NAMESPACE":   true,
	"PIGGY_POD_NAME":        true,
	"PIGGY_DEBUG":           true,
	"PIGGY_STANDALONE":      true,
	"PIGGY_ADDRESS":         true,
	"PIGGY_ALLOWED_SA":      true,
	"PIGGY_SKIP_VERIFY_TLS": true,
}
var schemeRegx = regexp.MustCompile(`piggy:(.+)`)

func (e *SanitizedEnv) append(name string, value string) {
	if _, ok := sanitizeEnvmap[name]; !ok {
		(*e)[name] = value
	}
}

func awsErr(err error) bool {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			log.Error().Err(aerr).Msg(aerr.Code())
		} else {
			log.Error().Err(aerr).Msg(err.Error())
		}
		return true
	}
	return false
}

func injectSecrets(config *PiggyConfig, references map[string]string, env *SanitizedEnv) {
	// Create a Secrets Manager client
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWSRegion),
	})
	if awsErr(err) {
		return
	}
	svc := secretsmanager.New(sess)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(config.AWSSecretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	result, err := svc.GetSecretValue(input)
	if awsErr(err) {
		return
	}

	// TODO add allowed-sa in secret env

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	if result.SecretString != nil {
		var secrets map[string]string
		json.Unmarshal([]byte(*result.SecretString), &secrets)
		if len(references) == 0 {
			// retrieve all secrets
			for name, value := range secrets {
				env.append(name, value)
			}
		} else {
			// filter by reference
			for refName, refValue := range references {
				if strings.HasPrefix(refValue, "piggy:") {
					match := schemeRegx.FindAllStringSubmatch(refValue, -1)
					if len(match) == 1 {
						if val, ok := secrets[match[0][1]]; ok {
							env.append(match[0][1], val)
						}
					}
					continue
				}
				env.append(refName, refValue)
			}
		}
	} else {
		// TODO a binary secret
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			fmt.Println("Base64 Decode Error:", err)
			return
		}
		decodedBinarySecret := string(decodedBinarySecretBytes[:len])
		log.Debug().Msgf("%v", decodedBinarySecret)
	}
}

func (s *Service) GetSecret(payload *GetSecretPayload) (*SanitizedEnv, error) {
	// creates the in-cluster config
	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	return nil, err
	// }
	// // creates the clientset
	// k8s, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	return nil, err
	// }
	// get a pod
	pod, err := s.k8sClient.CoreV1().Pods(payload.Namespace).Get(context.TODO(), payload.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("pod %s not found in %s namespace", payload.Name, payload.Namespace)
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			return nil, fmt.Errorf("error getting pod %v", statusError.ErrStatus.Message)
		}
		return nil, err
	}
	annotations := pod.Annotations
	config := &PiggyConfig{
		AWSSecretName: GetStringValue(annotations, AWSSecretName, ""),
		AWSRegion:     GetStringValue(annotations, ConfigAWSRegion, ""),
	}
	sanitized := &SanitizedEnv{}
	injectSecrets(config, map[string]string{}, sanitized)
	return sanitized, nil
}
