package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type sanitizedEnv struct {
	Env []string `json:"env"`
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
	"PIGGY_IGNORE_NO_ENV":   true,
}
var schemeRegx = regexp.MustCompile(`piggy:(.+)`)

func (e *sanitizedEnv) append(name string, value string) {
	if _, ok := sanitizeEnvmap[name]; !ok {
		e.Env = append(e.Env, fmt.Sprintf("%s=%s", name, value))
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

func injectSecrets(references map[string]string, env *sanitizedEnv) {
	secretName := os.Getenv("PIGGY_AWS_SECRET_NAME") // "exp/sample/test"
	region := os.Getenv("PIGGY_AWS_REGION")          // "ap-southeast-1"
	// secretName := "exp/sample/test"
	// region := "ap-southeast-1"

	// Create a Secrets Manager client
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if awsErr(err) {
		return
	}
	svc := secretsmanager.New(sess)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
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
		if err := json.Unmarshal([]byte(*result.SecretString), &secrets); err != nil {
			log.Error().Msgf("Error while unmarshal secret %v", err)
		}
		for refName, refValue := range references {
			if strings.HasPrefix(refValue, "piggy:") {
				match := schemeRegx.FindAllStringSubmatch(refValue, -1)
				if len(match) == 1 {
					if val, ok := secrets[match[0][1]]; ok {
						env.append(match[0][1], val)
						continue
					}
				}
			}
			env.append(refName, refValue)
		}
	} else {
		// TODO a binary secret
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			log.Error().Msgf("Base64 Decode Error: %v", err)
			return
		}
		decodedBinarySecret := string(decodedBinarySecretBytes[:len])
		log.Debug().Msgf("%v", decodedBinarySecret)
	}
}

type GetSecretPayload struct {
	Namespace string `json:"namespace"`
	Resources string `json:"resources"`
	Name      string `json:"name"`
	UID       string `json:"uid"`
	Signature string `json:"signature"`
}

func requestSecrets(references map[string]string, env *sanitizedEnv, sig []byte) {
	address := os.Getenv("PIGGY_ADDRESS")
	skipVerifyTLS := true
	if os.Getenv("PIGGY_SKIP_VERIFY_TLS") != "" {
		skipVerifyTLS, _ = strconv.ParseBool(os.Getenv("PIGGY_SKIP_VERIFY_TLS"))
	}

	log.Debug().Msgf("Address: %s", address)

	payload := GetSecretPayload{
		Namespace: os.Getenv("PIGGY_POD_NAMESPACE"),
		Name:      os.Getenv("PIGGY_POD_NAME"),
		Resources: "pods",
		UID:       os.Getenv("PIGGY_UID"),
		Signature: fmt.Sprintf("%x", sig),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		log.Error().Msgf("Invalid payload %v", err)
		return
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/secret", address), bytes.NewBuffer(b))
	if err != nil {
		log.Error().Msgf("Error while creating request %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerifyTLS},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Msgf("Error while requesting secret %v", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().Msgf("Error while reading secret %v", err)
	}
	var secrets map[string]string
	if err := json.Unmarshal(body, &secrets); err != nil {
		log.Error().Msgf("Error while translating secret %v", err)
	}

	for refName, refValue := range references {
		if strings.HasPrefix(refValue, "piggy:") {
			match := schemeRegx.FindAllStringSubmatch(refValue, -1)
			if len(match) == 1 {
				if val, ok := secrets[match[0][1]]; ok {
					env.append(match[0][1], val)
					continue
				}
			}
		}
		env.append(refName, refValue)
	}
}

func install(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	if fileInfo, err := os.Stat(dst); err == nil {
		if fileInfo.IsDir() {
			dst = dst + "/piggy-env"
		}
	}
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	buf := make([]byte, 1024)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	if err := os.Chmod(dst, 0700); err != nil {
		return err
	}
	return nil
}

// piggy-env install {location}
// piggy-env --standalone -- {command}
// piggy-env -- {command}
func main() {
	debug, _ := strconv.ParseBool(os.Getenv("PIGGY_DEBUG"))
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	args := os.Args
	var cmdArgs []string
	for i := range os.Args {
		if os.Args[i] == "--" {
			args = os.Args[:i]
			cmdArgs = os.Args[i+1:]
			break
		}
	}
	standalone := false
	if len(args) > 1 {
		if args[1] == "install" {
			if len(args) < 3 {
				log.Fatal().Msgf("Requires parameter")
			}
			err := install(args[0], args[2])
			if err != nil {
				log.Fatal().Msgf("Failed to install %v", err)
			}
			os.Exit(0)
		} else if args[1] == "--standalone" {
			standalone = true
		} else {
			log.Fatal().Msgf("Invalid command")
		}
	}
	// override arguments
	if os.Getenv("PIGGY_STANDALONE") != "" {
		standalone, _ = strconv.ParseBool(os.Getenv("PIGGY_STANDALONE"))
	}
	osEnv := make(map[string]string, len(os.Environ()))
	sanitized := sanitizedEnv{}
	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)
		name := split[0]
		value := split[1]
		osEnv[name] = value
	}
	if standalone {
		log.Debug().Msgf("Running in standalone mode")
		injectSecrets(osEnv, &sanitized)
	} else {
		log.Debug().Msgf("Running in lookup mode")
		sig := strings.TrimSpace(strings.Join(cmdArgs, " "))
		h := sha256.New()
		h.Write([]byte(sig))
		requestSecrets(osEnv, &sanitized, h.Sum(nil))
	}
	ignoreNoEnv := false
	if os.Getenv("PIGGY_IGNORE_NO_ENV") != "" {
		ignoreNoEnv, _ = strconv.ParseBool(os.Getenv("PIGGY_IGNORE_NO_ENV"))
	}
	if !ignoreNoEnv {
		for _, v := range sanitized.Env {
			split := strings.SplitN(v, "=", 2)
			if strings.HasPrefix(strings.ToUpper(split[1]), strings.ToUpper("piggy:")) {
				log.Fatal().Msgf("[%s] not found", split[0])
			}
		}
	}
	entrypointCmd := cmdArgs
	cmd, err := exec.LookPath(entrypointCmd[0])
	if err != nil {
		log.Fatal().Msgf("Command not found %s", entrypointCmd[0])
	}
	log.Debug().Msgf("spawning process: %s", entrypointCmd)
	err = syscall.Exec(cmd, entrypointCmd, sanitized.Env)
	if err != nil {
		log.Fatal().Msgf("failed to exec process %s [%s]", entrypointCmd, err.Error())
	}
}
