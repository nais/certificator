package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	CAUrls  []string `split_words:"true"`
	CAPaths []string `split_words:"true"`
}

const prefix = "CERTIFICATOR"

func NewFromEnv() (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process(prefix, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func Usage() error {
	return envconfig.Usage(prefix, &Config{})
	//return envconfig.Usagef(prefix, &Config{}, w, envconfig.DefaultTableFormat)
}
