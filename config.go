// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"os"
	"strconv"
	"time"
)

// Default exchange names and timeout, matching the platform and the Python SDK.
const (
	defaultRouterExchange = "griffino.router"
	defaultPluginExchange = "griffino.plugins"
	defaultInvokeTimeout  = 60 * time.Second
	defaultRabbitMQPort   = 5672
)

// Config holds the runtime configuration a plugin needs to talk to the
// platform's RabbitMQ broker. The platform injects the RabbitMQ credentials and
// the plugin identity into the container's environment; ConfigFromEnv reads them.
type Config struct {
	RabbitMQHost     string
	RabbitMQPort     int
	RabbitMQUser     string
	RabbitMQPassword string

	PluginID string
	UserID   string

	// InvokeTimeout is the default deadline applied to Invoke calls that do not
	// specify their own timeout.
	InvokeTimeout time.Duration

	// RouterExchange and PluginExchange are the topic exchanges used for
	// invoke/dispatch routing and per-plugin delivery respectively.
	RouterExchange string
	PluginExchange string
}

// withDefaults returns a copy of c with any zero-valued exchange names and
// timeout filled in.
func (c Config) withDefaults() Config {
	if c.RabbitMQPort == 0 {
		c.RabbitMQPort = defaultRabbitMQPort
	}
	if c.InvokeTimeout == 0 {
		c.InvokeTimeout = defaultInvokeTimeout
	}
	if c.RouterExchange == "" {
		c.RouterExchange = defaultRouterExchange
	}
	if c.PluginExchange == "" {
		c.PluginExchange = defaultPluginExchange
	}
	return c
}

// ConfigFromEnv builds a Config from the environment variables the platform
// injects: RABBITMQ_HOST, RABBITMQ_PORT (default 5672), RABBITMQ_USER,
// RABBITMQ_PASSWORD, GRIFFINO_PLUGIN_ID, and the optional GRIFFINO_USER_ID.
// It returns an error classified under ErrConfiguration when a required
// variable is missing or malformed.
func ConfigFromEnv() (Config, error) {
	host, err := requiredEnv("RABBITMQ_HOST")
	if err != nil {
		return Config{}, err
	}
	port, err := intEnv("RABBITMQ_PORT", defaultRabbitMQPort)
	if err != nil {
		return Config{}, err
	}
	user, err := requiredEnv("RABBITMQ_USER")
	if err != nil {
		return Config{}, err
	}
	password, err := requiredEnv("RABBITMQ_PASSWORD")
	if err != nil {
		return Config{}, err
	}
	pluginID, err := requiredEnv("GRIFFINO_PLUGIN_ID")
	if err != nil {
		return Config{}, err
	}
	return Config{
		RabbitMQHost:     host,
		RabbitMQPort:     port,
		RabbitMQUser:     user,
		RabbitMQPassword: password,
		PluginID:         pluginID,
		UserID:           os.Getenv("GRIFFINO_USER_ID"),
	}.withDefaults(), nil
}

// PlaceholderConfig returns a Config carrying only the plugin id, with empty
// broker credentials. It is used when a client is constructed solely to
// generate manifest/config artifacts and never connects to the broker.
func PlaceholderConfig(pluginID string) Config {
	return Config{PluginID: pluginID}.withDefaults()
}

// RedisConfig describes how to reach the Redis instance that backs per-user
// plugin configuration. It is optional: a plugin that does not read user config
// never needs it.
type RedisConfig struct {
	URL       string
	Host      string
	Port      int
	DB        int
	Username  string
	Password  string
	KeyPrefix string
}

const defaultRedisKeyPrefix = "user"

// RedisConfigFromEnv builds a RedisConfig from REDIS_URL / REDIS_HOST /
// REDIS_PORT / REDIS_DB / REDIS_USER / REDIS_PASSWORD and
// GRIFFINO_REDIS_USER_CONFIG_PREFIX. It returns ok=false when neither REDIS_URL
// nor REDIS_HOST is set, mirroring the Python SDK's optional behavior.
func RedisConfigFromEnv() (RedisConfig, bool, error) {
	url := os.Getenv("REDIS_URL")
	host := os.Getenv("REDIS_HOST")
	if url == "" && host == "" {
		return RedisConfig{}, false, nil
	}
	port, err := intEnv("REDIS_PORT", 6379)
	if err != nil {
		return RedisConfig{}, false, err
	}
	db, err := intEnv("REDIS_DB", 0)
	if err != nil {
		return RedisConfig{}, false, err
	}
	prefix := os.Getenv("GRIFFINO_REDIS_USER_CONFIG_PREFIX")
	if prefix == "" {
		prefix = defaultRedisKeyPrefix
	}
	if host == "" {
		host = "localhost"
	}
	return RedisConfig{
		URL:       url,
		Host:      host,
		Port:      port,
		DB:        db,
		Username:  os.Getenv("REDIS_USER"),
		Password:  os.Getenv("REDIS_PASSWORD"),
		KeyPrefix: prefix,
	}, true, nil
}

func requiredEnv(name string) (string, error) {
	if v := os.Getenv(name); v != "" {
		return v, nil
	}
	return "", configError("missing required environment variable: %s", name)
}

func intEnv(name string, def int) (int, error) {
	v := os.Getenv(name)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, configError("environment variable %s must be an integer", name)
	}
	return n, nil
}
