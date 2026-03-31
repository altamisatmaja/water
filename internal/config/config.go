package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	DBPath             string `mapstructure:"db_path" json:"db_path"`
	Host               string `mapstructure:"host" json:"host"`
	Port               int    `mapstructure:"port" json:"port"`
	EmbeddingMode      string `mapstructure:"embedding_mode" json:"embedding_mode"`
	ClaudeProjectsPath string `mapstructure:"claude_projects_path" json:"claude_projects_path,omitempty"`
	ClaudeProjectKey   string `mapstructure:"claude_project_key" json:"claude_project_key,omitempty"`
	ClaudeProjectPath  string `mapstructure:"claude_project_path" json:"claude_project_path,omitempty"`
	AnthropicAPIKey    string `mapstructure:"anthropic_api_key" json:"anthropic_api_key"`
	LogLevel           string `mapstructure:"log_level" json:"log_level"`
	EnableWebSocket    bool   `mapstructure:"enable_websocket" json:"enable_websocket"`
	EnableAnalytics    bool   `mapstructure:"enable_analytics" json:"enable_analytics"`
}

func LoadConfig(cfgPath string) (*Config, error) {
	v := viper.New()

	v.SetDefault("db_path", ".water")
	v.SetDefault("host", "127.0.0.1")
	v.SetDefault("port", 3141)
	v.SetDefault("embedding_mode", "local")
	v.SetDefault("claude_projects_path", "")
	v.SetDefault("claude_project_key", "")
	v.SetDefault("claude_project_path", "")
	v.SetDefault("log_level", "info")
	v.SetDefault("enable_websocket", true)
	v.SetDefault("enable_analytics", false)

	if cfgPath != "" {
		v.SetConfigFile(cfgPath)
		if err := v.ReadInConfig(); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	_ = v.BindEnv("anthropic_api_key", "ANTHROPIC_API_KEY")
	_ = v.BindEnv("db_path", "WATER_DB_PATH")
	_ = v.BindEnv("port", "WATER_PORT")
	_ = v.BindEnv("claude_projects_path", "WATER_CLAUDE_PROJECTS_PATH")
	_ = v.BindEnv("log_level", "WATER_LOG_LEVEL")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Save(cfgPath string) error {
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(cfgPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func GetConfigPath(dbPath string) string {
	return filepath.Join(dbPath, "config.json")
}

func GetEventsPath(dbPath string) string {
	return filepath.Join(dbPath, "events.jsonl")
}
