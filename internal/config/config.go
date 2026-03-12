package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Security SecurityConfig `yaml:"security"`
	Aliyun   AliyunConfig   `yaml:"aliyun"`
	Password string         `yaml:"password"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type SecurityConfig struct {
	MaxFailures   int           `yaml:"max_failures"`
	FailWindow    time.Duration `yaml:"fail_window"`
	BlockDuration time.Duration `yaml:"block_duration"`
}

type AliyunConfig struct {
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	RegionID        string `yaml:"region_id"`
	SecurityGroupID string `yaml:"security_group_id"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = "127.0.0.1:8080"
	}
	if cfg.Security.MaxFailures == 0 {
		cfg.Security.MaxFailures = 5
	}
	if cfg.Security.FailWindow == 0 {
		cfg.Security.FailWindow = 5 * time.Minute
	}
	if cfg.Security.BlockDuration == 0 {
		cfg.Security.BlockDuration = 30 * time.Minute
	}
	return &cfg, nil
}
