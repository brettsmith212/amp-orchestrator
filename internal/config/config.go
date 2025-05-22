package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Repository RepositoryConfig `mapstructure:"repository"`
	Agents     AgentConfig      `mapstructure:"agents"`
	Scheduler  SchedulerConfig  `mapstructure:"scheduler"`
	CI         CIConfig         `mapstructure:"ci"`
	IPC        IPCConfig        `mapstructure:"ipc"`
	Metrics    MetricsConfig    `mapstructure:"metrics"`
}

// RepositoryConfig holds git repository settings
type RepositoryConfig struct {
	Path    string `mapstructure:"path"`
	Workdir string `mapstructure:"workdir"`
}

// AgentConfig holds agent settings
type AgentConfig struct {
	Count   int `mapstructure:"count"`
	Timeout int `mapstructure:"timeout"`
}

// SchedulerConfig holds scheduler settings
type SchedulerConfig struct {
	PollInterval int    `mapstructure:"poll_interval"`
	BacklogPath  string `mapstructure:"backlog_path"`
	StaleTimeout int    `mapstructure:"stale_timeout"`
}

// CIConfig holds continuous integration settings
type CIConfig struct {
	StatusPath  string `mapstructure:"status_path"`
	QuickTests  bool   `mapstructure:"quick_tests"`
}

// IPCConfig holds inter-process communication settings
type IPCConfig struct {
	SocketPath string `mapstructure:"socket_path"`
}

// MetricsConfig holds metrics collection settings
type MetricsConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	OutputPath string `mapstructure:"output_path"`
}

// Load loads the configuration from file
func Load() (*Config, error) {
	v := viper.New()
	
	// Set defaults
	setDefaults(v)
	
	// Look for config in standard locations
	configFile := "config.yaml"
	configPaths := []string{
		".",
		"$HOME/.config/orchestrator",
		"/etc/orchestrator",
	}
	
	var configFound bool
	for _, path := range configPaths {
		expandedPath := os.ExpandEnv(path)
		v.AddConfigPath(expandedPath)
		
		// Check if file exists
		fullPath := filepath.Join(expandedPath, configFile)
		if _, err := os.Stat(fullPath); err == nil {
			configFound = true
			break
		}
	}
	
	if !configFound {
		return nil, errors.New("config file not found in any of the search paths")
	}
	
	v.SetConfigName(strings.TrimSuffix(configFile, filepath.Ext(configFile)))
	v.SetConfigType("yaml")
	
	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}
	
	// Validate the config
	if err := validateConfig(&config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Repository defaults
	v.SetDefault("repository.path", "./repo.git")
	v.SetDefault("repository.workdir", "./tmp")
	
	// Agent defaults
	v.SetDefault("agents.count", 3)
	v.SetDefault("agents.timeout", 1800) // 30 minutes
	
	// Scheduler defaults
	v.SetDefault("scheduler.poll_interval", 5)
	v.SetDefault("scheduler.backlog_path", "./backlog")
	v.SetDefault("scheduler.stale_timeout", 900) // 15 minutes
	
	// CI defaults
	v.SetDefault("ci.status_path", "./ci-status")
	v.SetDefault("ci.quick_tests", true)
	
	// IPC defaults
	v.SetDefault("ipc.socket_path", "~/.orchestrator.sock")
	
	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.output_path", "./metrics")
}

// validateConfig validates the loaded configuration
func validateConfig(config *Config) error {
	// Validate repository config
	if config.Repository.Path == "" {
		return errors.New("repository.path cannot be empty")
	}
	
	if config.Repository.Workdir == "" {
		return errors.New("repository.workdir cannot be empty")
	}
	
	// Validate agent config
	if config.Agents.Count < 1 {
		return errors.New("agents.count must be at least 1")
	}
	
	if config.Agents.Timeout < 60 {
		return errors.New("agents.timeout must be at least 60 seconds")
	}
	
	// Validate scheduler config
	if config.Scheduler.PollInterval < 1 {
		return errors.New("scheduler.poll_interval must be at least 1 second")
	}
	
	if config.Scheduler.BacklogPath == "" {
		return errors.New("scheduler.backlog_path cannot be empty")
	}
	
	return nil
}