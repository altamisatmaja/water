package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	DBPath          string `mapstructure:"db_path" json:"db_path"`
	Host            string `mapstructure:"host" json:"host"`
	Port            int    `mapstructure:"port" json:"port"`
	EmbeddingMode   string `mapstructure:"embedding_mode" json:"embedding_mode"`
	AnthropicAPIKey string `mapstructure:"anthropic_api_key" json:"anthropic_api_key"`
	LogLevel        string `mapstructure:"log_level" json:"log_level"`
	EnableWebSocket bool   `mapstructure:"enable_websocket" json:"enable_websocket"`
	EnableAnalytics bool   `mapstructure:"enable_analytics" json:"enable_analytics"`
}

func LoadConfig(cfgPath string) (*Config, error) {
	v := viper.New()

	v.SetDefault("db_path", ".water")
	v.SetDefault("host", "127.0.0.1")
	v.SetDefault("port", 3141)
	v.SetDefault("embedding_mode", "local")
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
	_ = v.BindEnv("log_level", "WATER_LOG_LEVEL")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Save(cfgPath string) error {
	v := viper.New()
	v.Set("db_path", c.DBPath)
	v.Set("host", c.Host)
	v.Set("port", c.Port)
	v.Set("embedding_mode", c.EmbeddingMode)
	v.Set("anthropic_api_key", c.AnthropicAPIKey)
	v.Set("log_level", c.LogLevel)
	v.Set("enable_websocket", c.EnableWebSocket)
	v.Set("enable_analytics", c.EnableAnalytics)

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	if err := v.WriteConfigAs(cfgPath); err != nil {
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
