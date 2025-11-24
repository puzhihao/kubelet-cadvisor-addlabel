package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config captures the runtime parameters for the collector.
type Config struct {
	Port               int    `json:"port" env:"PORT"`
	LogLevel           string `json:"log_level" env:"LOG_LEVEL"`
	AddLabels          string `json:"add_labels" env:"ADD_LABELS"`
	LabelDefaults      string `json:"label_defaults" env:"LABEL_DEFAULTS"`
	TokenFile          string `json:"token_file" env:"TOKEN_FILE"`
	CACertFile         string `json:"ca_cert_file" env:"CA_CERT_FILE"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify" env:"INSECURE_SKIP_VERIFY"`
	FetchInterval      int    `json:"fetch_interval" env:"FETCH_INTERVAL"`
}

// NewConfig loads configuration from environment variables, falling back to sensible defaults.
func NewConfig() *Config {
	return &Config{
		Port:               getEnvInt("PORT", 9090),
		LogLevel:           getEnvString("LOG_LEVEL", "info"),
		AddLabels:          getEnvString("ADD_LABELS", ""),
		LabelDefaults:      getEnvString("LABEL_DEFAULTS", "unknown"),
		TokenFile:          getEnvString("TOKEN_FILE", "/var/run/secrets/kubernetes.io/serviceaccount/token"),
		CACertFile:         getEnvString("CA_CERT_FILE", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"),
		InsecureSkipVerify: getEnvBool("INSECURE_SKIP_VERIFY", false),
		FetchInterval:      getEnvInt("FETCH_INTERVAL", 30),
	}
}

// Validate ensures the configuration values fall within acceptable ranges.
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be within range 1-65535")
	}

	if c.FetchInterval <= 0 {
		return fmt.Errorf("fetch interval must be greater than zero seconds")
	}

	return nil
}

// Verbosity returns the klog verbosity level to apply.
func (c *Config) Verbosity() int {
	switch strings.ToLower(strings.TrimSpace(c.LogLevel)) {
	case "error":
		return 0
	case "warn", "warning":
		return 1
	case "debug":
		return 4
	case "trace":
		return 6
	default:
		return 2 // info
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
