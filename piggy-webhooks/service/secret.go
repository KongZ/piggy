package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	authv1 "k8s.io/api/authentication/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/rs/zerolog/log"
)

type SanitizedEnv map[string]string

type GetSecretPayload struct {
	Resources string `json:"resources"`
	Name      string `json:"name"`
	UID       string `json:"uid"`
	Signature string `json:"signature"`
	Token     string `json:"-"`
}

type Info struct {
	Resources      string `json:"resources"`
	Name           string `json:"name"`
	UID            string `json:"uid"`
	Namespace      string `json:"namespace"`
	ServiceAccount string `json:"serviceAccount"`
}

type Signature map[string]string

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
	"PIGGY_POD_NAME":        true,
	"PIGGY_DEBUG":           true,
	"PIGGY_STANDALONE":      true,
	"PIGGY_ADDRESS":         true,
	"PIGGY_ALLOWED_SA":      true,
	"PIGGY_SKIP_VERIFY_TLS": true,
	"PIGGY_IGNORE_NO_ENV":   true,
}

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

func injectSecrets(config *PiggyConfig, env *SanitizedEnv) {
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

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	if result.SecretString != nil {
		var secrets map[string]string
		err := json.Unmarshal([]byte(*result.SecretString), &secrets)
		if err != nil {
			log.Error().Msgf("Unmashal error: %v", err)
			return
		}

		allowed := false
		if sas, ok := secrets["PIGGY_ALLOWED_SA"]; ok && config.PodServiceAccountName != "" {
			log.Debug().Msgf("Allowed service accounts [%s]", sas)
			log.Debug().Msgf("Pod service account [%s]", config.PodServiceAccountName)
			// if secrets contains PIGGY_ALLOWED_SA
			for _, sa := range strings.Split(sas, ",") {
				if sa == config.PodServiceAccountName {
					allowed = true
					break
				}
			}
		} else {
			allowed = !config.PiggyEnforceServiceAccount
		}
		log.Debug().Msgf("Decision [%v]", allowed)
		if allowed {
			for name, value := range secrets {
				env.append(name, value)
			}
		}
	} else {
		// TODO a binary secret
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			log.Error().Msgf("Base64 decode error: %v", err)
			return
		}
		decodedBinarySecret := string(decodedBinarySecretBytes[:len])
		log.Debug().Msgf("%v", decodedBinarySecret)
	}
}

func (s *Service) GetSecret(payload *GetSecretPayload) (*SanitizedEnv, Info, error) {
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
	info := Info{
		Resources: payload.Resources,
		Name:      payload.Name,
		UID:       payload.UID,
	}
	tr := authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: payload.Token,
		},
	}
	review, err := s.k8sClient.AuthenticationV1().TokenReviews().Create(context.TODO(), &tr, metav1.CreateOptions{})
	if err != nil {
		if statusError, isStatus := err.(*k8serrors.StatusError); isStatus {
			return nil, info, fmt.Errorf("error review token %v", statusError.ErrStatus.Message)
		}
		return nil, info, err
	}
	if !review.Status.Authenticated {
		return nil, info, errors.New("token is not authenticated")
	}
	fqSa := review.Status.User.Username
	tokenSa := strings.TrimPrefix(fqSa, "system:serviceaccount:")
	log.Debug().Msgf("request from [sa:%s], [pod:%s]", tokenSa, payload.Name)
	namespace := strings.Split(tokenSa, ":")[0]
	info.Namespace = namespace
	info.ServiceAccount = tokenSa
	// get a pod
	pod, err := s.k8sClient.CoreV1().Pods(namespace).Get(context.TODO(), payload.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, info, fmt.Errorf("pod %s not found in %s namespace", payload.Name, namespace)
		} else if statusError, isStatus := err.(*k8serrors.StatusError); isStatus {
			return nil, info, fmt.Errorf("error getting pod %v", statusError.ErrStatus.Message)
		}
		return nil, info, err
	}
	podSa := fmt.Sprintf("%s:%s", namespace, pod.Spec.ServiceAccountName)
	if podSa != tokenSa {
		return nil, info, fmt.Errorf("invalid service account found %s, expected %s", podSa, tokenSa)
	}
	annotations := pod.Annotations
	config := &PiggyConfig{
		AWSSecretName:              GetStringValue(annotations, AWSSecretName, ""),
		AWSRegion:                  GetStringValue(annotations, ConfigAWSRegion, ""),
		PodServiceAccountName:      tokenSa,
		PiggyEnforceIntegrity:      GetBoolValue(annotations, ConfigPiggyEnforceIntegrity, true),
		PiggyEnforceServiceAccount: GetBoolValue(EmptyMap, ConfigPiggyEnforceServiceAccount, false),
	}
	signature := make(Signature)
	if err := json.Unmarshal([]byte(annotations[Namespace+ConfigPiggyUID]), &signature); err != nil {
		log.Error().Msgf("Error while unmarshal signature %v", err)
	}
	if config.PiggyEnforceIntegrity {
		if signature[payload.UID] != payload.Signature {
			return nil, info, fmt.Errorf("%s invalid signature", payload.Name)
		}
	} else if signature[payload.UID] == "" {
		return nil, info, fmt.Errorf("%s invalid uid", payload.Name)
	}

	sanitized := &SanitizedEnv{}
	injectSecrets(config, sanitized)
	return sanitized, info, nil
}
