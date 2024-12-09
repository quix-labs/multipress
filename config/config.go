package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

type CredentialsConfig struct {
	DBName     string `yaml:"dbname,omitempty"`
	DBUser     string `yaml:"dbuser,omitempty"`
	DBPassword string `yaml:"dbpassword,omitempty"`

	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Email    string `yaml:"email,omitempty"`
}

type InstancesConfig struct {
	_counter    *atomic.Uint64               `yaml:"-"`
	Resources   ResourcesConfig              `yaml:"resources"`
	Credentials map[string]CredentialsConfig `yaml:"credentials"`
}

type ResourcesConfig struct {
	Memory string `yaml:"memory,omitempty"`
}

type MysqlConfig struct {
	Resources    ResourcesConfig `yaml:"resources,omitempty"`
	RootPassword string          `yaml:"root-password,omitempty"`
}

type ModelConfig struct {
	Resources   ResourcesConfig   `yaml:"resources,omitempty"`
	Credentials CredentialsConfig `yaml:"credentials,omitempty"`
}

type CaddyConfig struct {
	Resources ResourcesConfig `yaml:"resources,omitempty"`
	TLSIssuer string          `yaml:"tls-issuer,omitempty"`
}

type Config struct {
	Project    string `yaml:"project,omitempty"`
	BaseDomain string `yaml:"base-domain,omitempty"`
	Uid        int    `yaml:"uid,omitempty"`
	Gid        int    `yaml:"gid,omitempty"`

	Caddy     *CaddyConfig     `yaml:"caddy,omitempty"`
	MySql     *MysqlConfig     `yaml:"mysql,omitempty"`
	Model     *ModelConfig     `yaml:"model,omitempty"`
	Instances *InstancesConfig `yaml:"instances,omitempty"`
}

func (cfg *Config) VolumePath() string {
	return "./volumes" // Important keep ./ or use absolute
}

func (cfg *Config) BackupsPath() string {
	return "./backups" // Important keep ./ or use absolute
}

func (cfg *Config) NetworkName() string {
	return cfg.Project + "-network"
}

func (cfg *Config) CaddyContainerName() string {
	return cfg.Project + "-caddy"
}

func (cfg *Config) BackupsContainerName() string {
	return cfg.Project + "-backups"
}

func (cfg *Config) MysqlContainerName() string {
	return cfg.Project + "-mysql"
}

func (cfg *Config) PhpMyAdminContainerName() string {
	return cfg.Project + "-phpmyadmin"
}

func (cfg *Config) ModelContainerName() string {
	return cfg.Project + "-model"
}

func (cfg *Config) BackupsUrl() string {
	return "https://backups." + cfg.BaseDomain
}

func (cfg *Config) ModelUrl() string {
	return "https://model." + cfg.BaseDomain
}

func (cfg *Config) PhpMyAdminUrl() string {
	return "https://phpmyadmin." + cfg.BaseDomain
}

func (cfg *Config) InstanceUrl(identifier string) string {
	return "https://" + identifier + "." + cfg.BaseDomain
}

func (cfg *Config) InstanceVolumePath(identifier string) string {
	return cfg.VolumePath() + "/" + identifier
}

func (cfg *Config) ModelVolumePath() string {
	return cfg.VolumePath() + "/model"
}
func (cfg *Config) MysqlVolumePath() string {
	return cfg.VolumePath() + "/mysql"
}

func (cfg *Config) InstanceContainerName(identifier string) string {
	return cfg.Project + "-" + identifier
}

func (cfg *Config) SaveAs(path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file at %s: %w", path, err)
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data to file: %w", err)
	}

	return nil
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file at %s: %w", path, err)
	}
	defer file.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML data: %w", err)
	}

	return &cfg, nil
}

func (c *InstancesConfig) NextIdentifier() string {
	const instancePrefix = "user"

	// Initialize counter with introspected value if needed
	if c._counter == nil {
		var maxNumber uint64
		for key := range c.Credentials {
			if strings.HasPrefix(key, instancePrefix) {
				numberStr := strings.TrimPrefix(key, instancePrefix)
				number, err := strconv.Atoi(numberStr)
				if err == nil && uint64(number) > maxNumber {
					maxNumber = uint64(number)
				}
			}
		}
		c._counter = new(atomic.Uint64)
		c._counter.Store(maxNumber)
	}

	nextNumber := c._counter.Add(1)
	return fmt.Sprintf("%s%d", instancePrefix, nextNumber)
}
