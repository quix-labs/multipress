package config

import (
	"fmt"
	"github.com/quix-labs/multipress/utils"
	"syscall"
)

func NewDefaultConfig() *Config {
	return &Config{
		Project: "multipress",
		Uid:     syscall.Getuid(),
		Gid:     syscall.Getgid(),
	}
}
func NewDefaultCaddyConfig() *CaddyConfig {
	return &CaddyConfig{
		Resources: ResourcesConfig{
			Memory: "256M",
		},
		TLSIssuer: "internal",
	}
}

func NewDefaultMysqlConfig() *MysqlConfig {
	return &MysqlConfig{
		Resources: ResourcesConfig{
			Memory: "2G",
		},
		RootPassword: utils.GenerateSecurePassword(16),
	}
}

func NewDefaultModelConfig(cfg *Config) *ModelConfig {
	return &ModelConfig{
		Resources: ResourcesConfig{
			Memory: "512M",
		},
		Credentials: CredentialsConfig{
			DBName:     "model",
			DBUser:     "model",
			DBPassword: utils.GenerateSecurePassword(16),
			Username:   "admin",
			Password:   utils.GenerateSecurePassword(16),
			Email:      "admin@model." + cfg.BaseDomain,
		},
	}
}

func NewDefaultInstancesConfig(cfg *Config) *InstancesConfig {
	return &InstancesConfig{
		Resources: ResourcesConfig{
			Memory: "512M",
		},
		Credentials: make(map[string]CredentialsConfig),
	}
}

func NewDefaultInstanceCredentialConfig(cfg *Config, identifier string) *CredentialsConfig {
	return &CredentialsConfig{
		DBName:     identifier,
		DBUser:     identifier,
		DBPassword: utils.GenerateSecurePassword(16),

		Username: identifier,
		Password: utils.GenerateSecurePassword(16),
		Email:    fmt.Sprintf(`%s@%s.%s`, identifier, identifier, cfg.BaseDomain),
	}
}
