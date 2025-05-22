package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadDefaultConfig(t *testing.T) {
	// Setup a temporary directory for our test config
	tmpDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy sample config to temp directory
	sampleConfig, err := os.ReadFile("../../config.sample.yaml")
	if err != nil {
		t.Fatalf("Failed to read sample config: %v", err)
	}

	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, sampleConfig, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create a test viper instance
	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Set expected defaults
	setDefaults(v)

	// Unmarshal the config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify the config values match expected defaults
	if config.Repository.Path != "./repo.git" {
		t.Errorf("Expected repository.path to be './repo.git', got '%s'", config.Repository.Path)
	}

	if config.Repository.Workdir != "./tmp" {
		t.Errorf("Expected repository.workdir to be './tmp', got '%s'", config.Repository.Workdir)
	}

	if config.Agents.Count != 3 {
		t.Errorf("Expected agents.count to be 3, got %d", config.Agents.Count)
	}

	if config.Agents.Timeout != 1800 {
		t.Errorf("Expected agents.timeout to be 1800, got %d", config.Agents.Timeout)
	}

	if config.Scheduler.PollInterval != 5 {
		t.Errorf("Expected scheduler.poll_interval to be 5, got %d", config.Scheduler.PollInterval)
	}

	if config.Scheduler.BacklogPath != "./backlog" {
		t.Errorf("Expected scheduler.backlog_path to be './backlog', got '%s'", config.Scheduler.BacklogPath)
	}
}

func TestValidateConfig(t *testing.T) {
	// Test valid config
	validConfig := &Config{
		Repository: RepositoryConfig{
			Path:    "./repo.git",
			Workdir: "./tmp",
		},
		Agents: AgentConfig{
			Count:   3,
			Timeout: 1800,
		},
		Scheduler: SchedulerConfig{
			PollInterval: 5,
			BacklogPath:  "./backlog",
			StaleTimeout: 900,
		},
	}

	if err := validateConfig(validConfig); err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}

	// Test invalid repository path
	invalidRepoPath := *validConfig
	invalidRepoPath.Repository.Path = ""
	if err := validateConfig(&invalidRepoPath); err == nil {
		t.Error("Expected error for empty repository path, got nil")
	}

	// Test invalid agent count
	invalidAgentCount := *validConfig
	invalidAgentCount.Agents.Count = 0
	if err := validateConfig(&invalidAgentCount); err == nil {
		t.Error("Expected error for invalid agent count, got nil")
	}

	// Test invalid timeout
	invalidTimeout := *validConfig
	invalidTimeout.Agents.Timeout = 30
	if err := validateConfig(&invalidTimeout); err == nil {
		t.Error("Expected error for invalid timeout, got nil")
	}

	// Test invalid poll interval
	invalidPollInterval := *validConfig
	invalidPollInterval.Scheduler.PollInterval = 0
	if err := validateConfig(&invalidPollInterval); err == nil {
		t.Error("Expected error for invalid poll interval, got nil")
	}
}