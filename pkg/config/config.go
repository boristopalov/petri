package config

import (
	"time"
)

type ExperimentConfig struct {
	Name         string        `yaml:"name"`
	Duration     time.Duration `yaml:"duration"`
	StepInterval time.Duration `yaml:"step_interval"`
	Agents       []AgentConfig `yaml:"agents"`
	Environment  EnvConfig     `yaml:"environment"`
	Logging      LogConfig     `yaml:"logging"`
}

type LogConfig struct {
	Level   string   `yaml:"level"`
	Path    string   `yaml:"path"`
	Metrics []string `yaml:"metrics"`
}

type AgentConfig struct {
	Model  string         `yaml:"model"`
	Count  int            `yaml:"count"`
	Config map[string]any `yaml:"config"`
}

type EnvConfig struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:"config"`
}

func LoadConfig(path string) (*ExperimentConfig, error) {
	// TODO: Implement configuration loading from YAML file
	return nil, nil
}
