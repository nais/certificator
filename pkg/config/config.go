package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	CAUrls                []string      `split_words:"true"`
	CADirectories         []string      `split_words:"true"`
	DownloadTimeout       time.Duration `split_words:"true" default:"5s"`
	DownloadInterval      time.Duration `split_words:"true" default:"24h"`
	DownloadRetryInterval time.Duration `split_words:"true" default:"10m"`
	ApplyBackoff          time.Duration `split_words:"true" default:"5m"`
	ApplyTimeout          time.Duration `split_words:"true" default:"10s"`
	JksPassword           string        `split_words:"true" default:"changeme" required:"true"`
	LogFormat             LogFormat     `split_words:"true" default:"text" required:"true"`
	LogLevel              LogLevel      `split_words:"true" default:"debug" required:"true"`
	MetricsAddress        string        `split_words:"true" default:"127.0.0.1:8080"`
}

type LogFormat struct {
	Formatter log.Formatter
}

type LogLevel log.Level

func (format *LogFormat) Decode(value string) error {
	switch value {
	case "text":
		format.Formatter = &log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		}
	case "json":
		format.Formatter = &log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		}
	default:
		return fmt.Errorf("unsupported log format %q, expected %q or %q", value, "text", "json")
	}
	return nil
}

func (loglevel *LogLevel) Decode(value string) error {
	lvl, err := log.ParseLevel(value)
	*loglevel = LogLevel(lvl)
	return err
}

const prefix = "CERTIFICATOR"

func NewFromEnv() (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process(prefix, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, cfg.Validate()
}

func (cfg *Config) Validate() error {
	if len(cfg.CAUrls)+len(cfg.CADirectories) == 0 {
		return fmt.Errorf("no CA certificate sources configured")
	}
	for i, path := range cfg.CADirectories {
		path, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		cfg.CADirectories[i] = path
		stat, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
	}
	return nil
}

func Usage() error {
	return envconfig.Usage(prefix, &Config{})
	//return envconfig.Usagef(prefix, &Config{}, w, envconfig.DefaultTableFormat)
}
