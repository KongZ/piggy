package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/smithy-go"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const PrefixPiggy = "piggy:"

type sanitizedEnv struct {
	Env []string `json:"env"`
}

var sanitizeEnvmap = map[string]bool{
	"PIGGY_AWS_SECRET_NAME":            true,
	"PIGGY_AWS_SSM_PARAMETER_PATH":     true,
	"PIGGY_AWS_REGION":                 true,
	"PIGGY_POD_NAME":                   true,
	"PIGGY_DEBUG":                      true,
	"PIGGY_STANDALONE":                 true,
	"PIGGY_ADDRESS":                    true,
	"PIGGY_ALLOWED_SA":                 true,
	"PIGGY_SKIP_VERIFY_TLS":            true,
	"PIGGY_IGNORE_NO_ENV":              true,
	"PIGGY_DEFAULT_SECRET_NAME_PREFIX": true, // use before secret
	"PIGGY_DEFAULT_SECRET_NAME_SUFFIX": true, // use before secret
	"PIGGY_DNS_RESOLVER":               true, // use before secret
	"PIGGY_INITIAL_DELAY":              true, // use before secret
	"PIGGY_NUMBER_OF_RETRY":            true, // use before secret
}

var golangNetwork = map[string]bool{
	"tcp":        true,
	"tcp4":       true,
	"tcp6":       true,
	"udp":        true,
	"udp4":       true,
	"udp6":       true,
	"ip":         true,
	"ip4":        true,
	"ip6":        true,
	"unix":       true,
	"unixgram":   true,
	"unixpacket": true,
}

var schemeRegx = regexp.MustCompile(`piggy:(.+)`)

func (e *sanitizedEnv) append(name string, value string) {
	if _, ok := sanitizeEnvmap[name]; !ok {
		e.Env = append(e.Env, fmt.Sprintf("%s=%s", name, value))
	}
}

func awsErr(err error) bool {
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			log.Error().Err(apiErr).Msgf("[%s] %s", apiErr.ErrorCode(), apiErr.ErrorMessage())
		}
		return true
	}
	return false
}

func doSanitize(references map[string]string, env *sanitizedEnv, secrets map[string]string) {
	for refName, refValue := range references {
		if strings.HasPrefix(refValue, PrefixPiggy) {
			match := schemeRegx.FindAllStringSubmatch(refValue, -1)
			if len(match) == 1 {
				if val, ok := secrets[match[0][1]]; ok {
					env.append(refName, val)
					continue
				}
			}
		}
		env.append(refName, refValue)
	}
}

func inject(references map[string]string, env *sanitizedEnv) error {
	ssmPath := os.Getenv("PIGGY_AWS_SSM_PARAMETER_PATH")
	if ssmPath == "" {
		return injectSecrets(references, env)
	}
	return injectParameters(references, env)
}

func injectParameters(references map[string]string, env *sanitizedEnv) error {
	ssmPath := os.Getenv("PIGGY_AWS_SSM_PARAMETER_PATH") // "/exp/sample/test"
	region := os.Getenv("PIGGY_AWS_REGION")              // "ap-southeast-1"
	// Create a SSM client
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return err
	}
	pm := ssm.NewFromConfig(cfg)
	// Get parameter values
	var nextToken *string
	secrets := make(map[string]string)
	for {
		input := &ssm.GetParametersByPathInput{
			Path:           aws.String(ssmPath),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true),
			MaxResults:     aws.Int32(10),
			NextToken:      nextToken,
		}
		output, err := pm.GetParametersByPath(context.TODO(), input)
		if awsErr(err) {
			return err
		}
		for _, param := range output.Parameters {
			name := filepath.Base(*param.Name)
			secrets[name] = *param.Value
		}
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}
	doSanitize(references, env, secrets)
	return nil
}

func injectSecrets(references map[string]string, env *sanitizedEnv) error {
	secretName := os.Getenv("PIGGY_AWS_SECRET_NAME") // "exp/sample/test"
	region := os.Getenv("PIGGY_AWS_REGION")          // "ap-southeast-1"
	// secretName := "exp/sample/test"
	// region := "ap-southeast-1"

	// Create a Secrets Manager client
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return err
	}
	sm := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}
	output, err := sm.GetSecretValue(context.TODO(), input)
	if awsErr(err) {
		return err
	}

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	if output.SecretString != nil {
		var secrets map[string]string
		if err := json.Unmarshal([]byte(*output.SecretString), &secrets); err != nil {
			log.Error().Msgf("Error while unmarshal secret %v", err)
		}
		doSanitize(references, env, secrets)
	} else {
		log.Info().Msgf("A binary secret is not supported")
		// decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(output.SecretBinary)))
		// len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, output.SecretBinary)
		// if err != nil {
		// 	log.Error().Msgf("Base64 Decode Error: %v", err)
		// 	return
		// }
		// decodedBinarySecret := string(decodedBinarySecretBytes[:len])
	}
	return nil
}

type GetSecretPayload struct {
	Resources string `json:"resources"`
	Name      string `json:"name"`
	UID       string `json:"uid"`
	Signature string `json:"signature"`
}

func requestSecrets(references map[string]string, env *sanitizedEnv, sig []byte) error {
	address := os.Getenv("PIGGY_ADDRESS")
	skipVerifyTLS := true
	if os.Getenv("PIGGY_SKIP_VERIFY_TLS") != "" {
		skipVerifyTLS, _ = strconv.ParseBool(os.Getenv("PIGGY_SKIP_VERIFY_TLS"))
	}

	log.Debug().Msgf("Address: %s", address)

	var serviceToken string
	b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return fmt.Errorf("failed to get token %v", err)
	}
	serviceToken = string(b)

	payload := GetSecretPayload{
		Name:      os.Getenv("PIGGY_POD_NAME"),
		Resources: "pods",
		UID:       os.Getenv("PIGGY_UID"),
		Signature: fmt.Sprintf("%x", sig),
	}
	b, err = json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("invalid payload %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/secret", address), bytes.NewBuffer(b))
	req.Header.Add("X-Token", serviceToken)
	if err != nil {
		return fmt.Errorf("error while creating request %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	tr := &http.Transport{
		// #nosec G402 possible self-sign
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerifyTLS},
	}
	client := &http.Client{Transport: tr}
	dnsResolver := os.Getenv("PIGGY_DNS_RESOLVER")
	if _, ok := golangNetwork[dnsResolver]; ok {
		log.Info().Msgf("Using DNS Resolver %s", dnsResolver)
		dialer := &net.Dialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{}
					return d.DialContext(ctx, dnsResolver, address)
				},
			},
		}
		client.Transport.(*http.Transport).DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := tls.DialWithDialer(dialer, network, addr, client.Transport.(*http.Transport).TLSClientConfig)
			if err != nil {
				return nil, err
			}
			return c, c.Handshake()
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error while requesting secret %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error while reading secret %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error while requesting secret %v", string(body))
	}
	var secrets map[string]string
	if err := json.Unmarshal(body, &secrets); err != nil {
		return fmt.Errorf("error while translating secret %v", err)
	}
	doSanitize(references, env, secrets)
	return nil
}

func install(src, dst string) error {
	source, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer func() {
		if err := source.Close(); err != nil {
			log.Error().Msgf("error closing file: %s\n", err)
		}
	}()
	if fileInfo, err := os.Stat(dst); err == nil {
		if fileInfo.IsDir() {
			dst = dst + "/piggy-env"
		}
	}
	destination, err := os.Create(filepath.Clean(dst))
	if err != nil {
		return err
	}
	defer func() {
		if err := destination.Close(); err != nil {
			log.Error().Msgf("Error closing file: %s\n", err)
		}
	}()
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
	// #nosec G302 we need piggy-env to executable
	if err := os.Chmod(dst, 0777); err != nil {
		return err
	}
	return nil
}

// piggy-env install {location}
// piggy-env {flag} -- {command}
//
//	flag:
//	  --standalone
//	  --initial-delay
//	  --retry
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
	numberOfRetry := 1
	initialDelay := time.Duration(0)
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
		}
		for i := 0; i < len(args); i++ {
			if args[i] == "--standalone" {
				standalone = true
			} else if args[i] == "--retry" {
				var i64 int64
				var err error
				i = i + 1
				if i >= len(args) {
					log.Fatal().Msg("Missing --retry argument value")
				}
				if i64, err = strconv.ParseInt(args[i], 10, 0); err != nil || i64 < 1 {
					log.Fatal().Msgf("Invalid --retry value. Expecting integer > 0")
				}
				numberOfRetry = int(i64)
			} else if args[i] == "--initial-delay" {
				var err error
				i = i + 1
				if i >= len(args) {
					log.Fatal().Msg("Missing --initial-delay argument value")
				}
				if initialDelay, err = time.ParseDuration(args[i]); err != nil {
					log.Fatal().Msgf("Invalid --initial-delay value. [%s]", err)
				}
			}
		}
	}

	if len(cmdArgs) == 0 {
		log.Fatal().Msgf("Invalid command")
	}
	// override arguments
	if os.Getenv("PIGGY_STANDALONE") != "" {
		standalone, _ = strconv.ParseBool(os.Getenv("PIGGY_STANDALONE"))
	}
	if os.Getenv("PIGGY_INITIAL_DELAY") != "" {
		var err error
		if initialDelay, err = time.ParseDuration(os.Getenv("PIGGY_INITIAL_DELAY")); err != nil {
			log.Info().Msgf("Invalid PIGGY_INITIAL_DELAY value. [%s]", err)
		}
	}
	if os.Getenv("PIGGY_NUMBER_OF_RETRY") != "" {
		var i64 int64
		var err error
		if i64, err = strconv.ParseInt(os.Getenv("PIGGY_NUMBER_OF_RETRY"), 10, 0); err != nil || i64 < 1 {
			log.Info().Msgf("Invalid PIGGY_NUMBER_OF_RETRY value. Expecting integer > 0")
		}
		numberOfRetry = int(i64)
	}

	// start the piggy
	osEnv := make(map[string]string, len(os.Environ()))
	sanitized := sanitizedEnv{}
	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)
		name := split[0]
		value := split[1]
		osEnv[name] = value
	}
	if initialDelay > 0 {
		log.Info().Msgf("Sleeping for %s", initialDelay)
		time.Sleep(initialDelay)
	}
	ignoreNoEnv := false
	if os.Getenv("PIGGY_IGNORE_NO_ENV") != "" {
		ignoreNoEnv, _ = strconv.ParseBool(os.Getenv("PIGGY_IGNORE_NO_ENV"))
	}
	retryResults := make([]string, numberOfRetry)
	success := false
	if standalone {
		log.Debug().Msgf("Running in standalone mode")
		for i := 0; !success && i < numberOfRetry; i++ {
			log.Debug().Msgf("Retry %d/%d", (i + 1), numberOfRetry)
			if e := inject(osEnv, &sanitized); e != nil {
				retryResults[i] = fmt.Sprintf("Retry %d/%d [error=%s]", (i + 1), numberOfRetry, e.Error())
				time.Sleep(500 * time.Millisecond)
			} else {
				log.Info().Msg("Request secrets was successful")
				break
			}
		}
	} else {
		log.Debug().Msgf("Running in proxy mode")
		sig := strings.TrimSpace(strings.Join(cmdArgs, " "))
		h := sha256.New()
		_, err := h.Write([]byte(sig))
		if err != nil {
			log.Error().Msgf("%v", err)
		}
		sum := h.Sum(nil)
		for i := 0; !success && i < numberOfRetry; i++ {
			log.Debug().Msgf("Retry %d/%d", (i + 1), numberOfRetry)
			if e := requestSecrets(osEnv, &sanitized, sum); e != nil {
				retryResults[i] = fmt.Sprintf("Retry %d/%d [error=%s]", (i + 1), numberOfRetry, e.Error())
				time.Sleep(500 * time.Millisecond)
			} else {
				success = true
				log.Info().Msg("Request secrets was successful")
				break
			}
		}
	}
	if !success {
		for _, result := range retryResults {
			log.Error().Msg(result)
		}
		if !ignoreNoEnv {
			log.Fatal().Msgf("Unable to communicate with %s", os.Getenv("PIGGY_ADDRESS"))
		}
	}
	if !ignoreNoEnv {
		for _, v := range sanitized.Env {
			split := strings.SplitN(v, "=", 2)
			if strings.HasPrefix(strings.ToUpper(split[1]), strings.ToUpper(PrefixPiggy)) {
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
	// #nosec G204 we intend to pass env to sub-process
	err = syscall.Exec(cmd, entrypointCmd, sanitized.Env)
	if err != nil {
		log.Fatal().Msgf("failed to exec process %s [%s]", entrypointCmd, err.Error())
	}
}
