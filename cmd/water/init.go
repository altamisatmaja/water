package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/water-viz/water/internal/config"
	"github.com/water-viz/water/internal/graph"
	"github.com/water-viz/water/internal/logger"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .water/ (DuckDB + config + events log)",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, _ := cmd.Flags().GetString("db-path")
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		embeddingMode, _ := cmd.Flags().GetString("embedding-mode")

		if err := os.MkdirAll(dbPath, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dbPath, err)
		}

		eventsPath := config.GetEventsPath(dbPath)
		if _, err := os.OpenFile(eventsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err != nil {
			return fmt.Errorf("create events.jsonl: %w", err)
		}

		cfg := &config.Config{
			DBPath:          dbPath,
			Host:            host,
			Port:            port,
			EmbeddingMode:   embeddingMode,
			LogLevel:        "info",
			EnableWebSocket: true,
			EnableAnalytics: false,
		}
		cfgPath := config.GetConfigPath(dbPath)
		if err := cfg.Save(cfgPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		// Initialize DuckDB schema by opening a client once.
		c, err := graph.NewClient(cmd.Context(), dbPath)
		if err != nil {
			return fmt.Errorf("init duckdb: %w", err)
		}
		_ = c.Close()

		gitignorePath := filepath.Join(dbPath, ".gitignore")
		_ = os.WriteFile(gitignorePath, []byte("database.duckdb\ndatabase.duckdb.wal\nevents.jsonl\n"), 0o644)

		logger.Info("initialized", "db_path", dbPath)
		fmt.Printf("✓ Water initialized at %s\n", dbPath)
		fmt.Printf("  Next: water serve --db-path %s\n", dbPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("db-path", ".water", "Path to .water directory")
	initCmd.Flags().String("host", "127.0.0.1", "Bind host")
	initCmd.Flags().Int("port", 3141, "HTTP port")
	initCmd.Flags().String("embedding-mode", "local", "Embedding mode: local|api")
}
