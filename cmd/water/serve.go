package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/water-viz/water/internal/capture"
	"github.com/water-viz/water/internal/config"
	"github.com/water-viz/water/internal/graph"
	"github.com/water-viz/water/internal/logger"
	"github.com/water-viz/water/internal/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start Water HTTP server + dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, _ := cmd.Flags().GetString("db-path")
		openBrowser, _ := cmd.Flags().GetBool("open-browser")

		cfg, err := config.LoadConfig(config.GetConfigPath(dbPath))
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		cfg.DBPath = dbPath

		// Allow overrides from flags.
		if host, _ := cmd.Flags().GetString("host"); host != "" {
			cfg.Host = host
		}
		if port, _ := cmd.Flags().GetInt("port"); port != 0 {
			cfg.Port = port
		}

		ctx := context.Background()
		g, err := graph.NewClient(ctx, dbPath)
		if err != nil {
			return fmt.Errorf("open graph: %w", err)
		}
		defer g.Close()

		eventsPath := config.GetEventsPath(dbPath)
		w, err := capture.NewWriter(eventsPath)
		if err != nil {
			return fmt.Errorf("open events writer: %w", err)
		}
		defer w.Close()

		srv := server.NewServer(cfg, g, w, eventsPath)

		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		url := fmt.Sprintf("http://%s", addr)
		logger.Info("serve", "addr", addr)

		if openBrowser {
			go openURL(url)
		}

		return http.ListenAndServe(addr, srv.Router())
	},
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	_ = cmd.Start()
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("db-path", ".water", "Path to .water directory")
	serveCmd.Flags().String("host", "127.0.0.1", "Bind host (overrides config)")
	serveCmd.Flags().Int("port", 3141, "HTTP port (overrides config)")
	serveCmd.Flags().Bool("open-browser", true, "Auto-open browser")
}
