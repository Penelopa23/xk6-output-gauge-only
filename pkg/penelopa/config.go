package penelopa

import (
	"crypto/tls"
	"encoding/json"
	_ "errors"
	"fmt"
	_ "net/http"
	"strconv"
	"strings"
	"time"
	"xk6-output-penelopa/pkg/remote"

	"go.k6.io/k6/lib/types"
	"gopkg.in/guregu/null.v3"
)

const (
	defaultServerURL    = "http://vms-victoria-metrics-single-victoria-server.metricstest:8428/api/v1/write"
	defaultTimeout      = 5 * time.Second
	defaultPushInterval = 5 * time.Second
	defaultBatchSize    = 1000
	defaultPod          = "PenelopaPod"
	defaultTestId       = "PenelopaTestId"
	defaultMetricPrefix = "k6_"
)

// Config contains the configuration for the Output.
type Config struct {
	// ServerURL contains the absolute ServerURL for the Write endpoint where to flush the time series.
	ServerURL null.String `json:"url"`

	// Headers contains additional headers that should be included in the HTTP requests.
	Headers map[string]string `json:"headers"`

	// InsecureSkipTLSVerify skips TLS client side checks.
	InsecureSkipTLSVerify null.Bool `json:"insecureSkipTLSVerify"`

	// PushInterval defines the time between flushes. The Output will wait the set time
	// before push a new set of time series to the endpoint.
	PushInterval types.NullDuration `json:"pushInterval"`

	TestId null.String `json:"testId"`

	Pod null.String `json:"pod"`

	BatchSize null.Int `json:"batchSize"`
}

// NewConfig creates an Output's configuration.
func NewConfig() Config {
	return Config{
		ServerURL:             null.StringFrom(defaultServerURL),
		InsecureSkipTLSVerify: null.BoolFrom(false),
		PushInterval:          types.NullDurationFrom(defaultPushInterval),
		Headers:               make(map[string]string),
		TestId:                null.StringFrom(defaultTestId),
		Pod:                   null.StringFrom(defaultPod),
		BatchSize:             null.IntFrom(defaultBatchSize),
	}
}

// RemoteConfig creates a configuration for the HTTP Remote-write client.
func (conf Config) RemoteConfig() (*remote.HTTPConfig, error) {
	hc := remote.HTTPConfig{
		Timeout: defaultTimeout,
	}

	hc.TLSConfig = &tls.Config{
		InsecureSkipVerify: conf.InsecureSkipTLSVerify.Bool, //nolint:gosec
	}

	return &hc, nil
}

// Apply merges applied Config into base.
func (conf Config) Apply(applied Config) Config {
	if applied.ServerURL.Valid {
		conf.ServerURL = applied.ServerURL
	}

	if applied.InsecureSkipTLSVerify.Valid {
		conf.InsecureSkipTLSVerify = applied.InsecureSkipTLSVerify
	}

	if applied.PushInterval.Valid {
		conf.PushInterval = applied.PushInterval
	}

	if applied.TestId.Valid {
		conf.TestId = applied.TestId
	}

	if applied.Pod.Valid {
		conf.Pod = applied.Pod
	}

	if applied.BatchSize.Valid {
		conf.BatchSize = applied.BatchSize
	}

	if len(applied.Headers) > 0 {
		for k, v := range applied.Headers {
			conf.Headers[k] = v
		}
	}

	return conf
}

// GetConsolidatedConfig combines the options' values from the different sources
// and returns the merged options. The Order of precedence used is documented
// in the k6 Documentation https://k6.io/docs/using-k6/k6-options/how-to/#order-of-precedence.
func GetConsolidatedConfig(jsonRawConf json.RawMessage, env map[string]string, _ string) (Config, error) {
	result := NewConfig()
	if jsonRawConf != nil {
		jsonConf, err := parseJSON(jsonRawConf)
		if err != nil {
			return result, fmt.Errorf("parse JSON options failed: %w", err)
		}
		result = result.Apply(jsonConf)
	}

	if len(env) > 0 {
		envConf, err := parseEnvs(env)
		if err != nil {
			return result, fmt.Errorf("parse environment variables options failed: %w", err)
		}
		result = result.Apply(envConf)
	}

	return result, nil
}

func envBool(env map[string]string, name string) (null.Bool, error) {
	if v, vDefined := env[name]; vDefined {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return null.NewBool(false, false), err
		}

		return null.BoolFrom(b), nil
	}
	return null.NewBool(false, false), nil
}

func envMap(env map[string]string, prefix string) map[string]string {
	result := make(map[string]string)
	for ek, ev := range env {
		if strings.HasPrefix(ek, prefix) {
			k := strings.TrimPrefix(ek, prefix)
			result[k] = ev
		}
	}
	return result
}

// TODO: try to migrate to github.com/mstoykov/envconfig like it's done on other projects?
func parseEnvs(env map[string]string) (Config, error) { //nolint:funlen
	c := Config{
		Headers: make(map[string]string),
	}

	if pushInterval, pushIntervalDefined := env["PENELOPA_METRICS_PUSH_INTERVAL"]; pushIntervalDefined {
		if err := c.PushInterval.UnmarshalText([]byte(pushInterval)); err != nil {
			return c, err
		}
	}

	if url, urlDefined := env["PENELOPA_METRICS_URL"]; urlDefined {
		c.ServerURL = null.StringFrom(url)
	}

	if testid, testIdDefined := env["PENELOPA_TESTID"]; testIdDefined {
		c.TestId = null.StringFrom(testid)
	}

	if pod, podDefined := env["PENELOPA_POD"]; podDefined {
		c.Pod = null.StringFrom(pod)
	}

	if batchSize, batchSizeDefined := env["PENELOPA_BATCH_SIZE"]; batchSizeDefined {
		if i, err := strconv.Atoi(batchSize); err == nil {
			c.BatchSize = null.IntFrom(int64(i))
		} else {
			return c, fmt.Errorf("invalid BATCH_SIZE: %w", err)
		}
	}

	if b, err := envBool(env, "PENELOPA_INSECURE_SKIP_TLS_VERIFY"); err != nil {
		return c, err
	} else if b.Valid {
		c.InsecureSkipTLSVerify = b
	}

	envHeaders := envMap(env, "HEADERS")
	for k, v := range envHeaders {
		c.Headers[k] = v
	}

	if headers, headersDefined := env["PENELOPA_HEADERS"]; headersDefined {
		for _, kvPair := range strings.Split(headers, ",") {
			header := strings.Split(kvPair, ":")
			if len(header) != 2 {
				return c, fmt.Errorf("the provided header (%s) does not respect the expected format <header key>:<value>", kvPair)
			}
			c.Headers[header[0]] = header[1]
		}
	}

	return c, nil
}

// parseJSON parses the supplied JSON into a Config.
func parseJSON(data json.RawMessage) (Config, error) {
	var c Config
	err := json.Unmarshal(data, &c)
	return c, err
}

// parseArg parses the supplied string of arguments into a Config.
func parseArg(text string) (Config, error) {
	var c Config
	opts := strings.Split(text, ",")

	for _, opt := range opts {
		r := strings.SplitN(opt, "=", 2)
		if len(r) != 2 {
			return c, fmt.Errorf("couldn't parse argument %q as option", opt)
		}
		key, v := r[0], r[1]
		switch key {
		case "url":
			c.ServerURL = null.StringFrom(v)
		case "insecureSkipTLSVerify":
			if err := c.InsecureSkipTLSVerify.UnmarshalText([]byte(v)); err != nil {
				return c, fmt.Errorf("insecureSkipTLSVerify value must be true or false, not %q", v)
			}
		default:
			if !strings.HasPrefix(key, "headers.") {
				return c, fmt.Errorf("%q is an unknown option's key", r[0])
			}
			if c.Headers == nil {
				c.Headers = make(map[string]string)
			}
			c.Headers[strings.TrimPrefix(key, "headers.")] = v
		}
	}

	return c, nil
}
