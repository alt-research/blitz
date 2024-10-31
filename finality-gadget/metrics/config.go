package metrics

import (
	"fmt"
	"net"
	"time"

	"github.com/alt-research/blitz/finality-gadget/core/utils"
)

const (
	DefaultFpMetricsPort         = 2112
	defaultMetricsHost           = "127.0.0.1"
	defaultMetricsUpdateInterval = 100 * time.Millisecond
)

type Config struct {
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	UpdateInterval time.Duration `yaml:"updateinterval"`
}

func (c *Config) WithEnv() {
	c.Host = utils.LookupEnvStr("FINALITY_GADGET_METRICS_HOST", c.Host)
	c.Port = int(utils.LookupEnvUint64("FINALITY_GADGET_METRICS_PORT", uint64(c.Port)))
}

func (cfg *Config) Validate() error {
	if cfg.Port < 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}

	ip := net.ParseIP(cfg.Host)
	if ip == nil {
		return fmt.Errorf("invalid host: %v", cfg.Host)
	}

	return nil
}

func (cfg *Config) Address() (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), nil
}

func DefaultFpConfig() *Config {
	return &Config{
		Port:           DefaultFpMetricsPort,
		Host:           defaultMetricsHost,
		UpdateInterval: defaultMetricsUpdateInterval,
	}
}
