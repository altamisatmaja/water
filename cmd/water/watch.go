package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/water-viz/water/internal/capture"
	"github.com/water-viz/water/internal/config"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Tail .water/events.jsonl in the terminal",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, _ := cmd.Flags().GetString("db-path")
		eventsPath := config.GetEventsPath(dbPath)

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		out := make(chan *capture.Event, 100)
		go func() {
			_ = capture.Tail(ctx, eventsPath, out)
			close(out)
		}()

		enc := json.NewEncoder(os.Stdout)
		for evt := range out {
			if err := enc.Encode(evt); err != nil {
				return fmt.Errorf("print event: %w", err)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().String("db-path", ".water", "Path to .water directory")
}
