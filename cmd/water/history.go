package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/water-viz/water/internal/claude"
	"github.com/water-viz/water/internal/config"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show Claude session history for the linked project",
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshot, err := loadClaudeSnapshotForCommand(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 || limit > len(snapshot.Sessions) {
			limit = len(snapshot.Sessions)
		}

		fmt.Printf("%s\n", snapshot.Project.Name)
		fmt.Printf("Claude project: %s\n", snapshot.Project.Path)
		fmt.Printf("Sessions: %d\n\n", snapshot.Stats.Sessions)

		for i, session := range snapshot.Sessions[:limit] {
			fmt.Printf("%d. %s  %s\n", i+1, session.ID, session.UpdatedAt.Format("2006-01-02 15:04"))
			if session.GitBranch != "" || session.CWD != "" {
				fmt.Printf("   %s", session.GitBranch)
				if session.CWD != "" {
					if session.GitBranch != "" {
						fmt.Print("  ")
					}
					fmt.Printf("%s", session.CWD)
				}
				fmt.Println()
			}
			fmt.Printf("   prompt: %s\n", fallback(session.PromptPreview, "(no prompt preview)"))
			if len(session.ToolNames) > 0 {
				fmt.Printf("   tools: %s\n", strings.Join(session.ToolNames, ", "))
			}
			fmt.Printf("   messages: %d  subagents: %d  paths: %d\n", session.MessageCount, len(session.Subagents), len(session.PathRefs))
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().String("db-path", ".water", "Path to .water directory")
	historyCmd.Flags().String("project", "", "Claude project name, key, or path")
	historyCmd.Flags().String("claude-projects-path", "", "Path to ~/.claude/projects")
	historyCmd.Flags().Int("limit", 10, "Maximum number of sessions to print")
}

func loadClaudeSnapshotForCommand(cmd *cobra.Command) (*claude.ProjectSnapshot, error) {
	dbPath, _ := cmd.Flags().GetString("db-path")
	projectQuery, _ := cmd.Flags().GetString("project")
	claudeProjectsPath, _ := cmd.Flags().GetString("claude-projects-path")

	cfg, _ := config.LoadConfig(config.GetConfigPath(dbPath))
	if claudeProjectsPath == "" && cfg != nil && cfg.ClaudeProjectsPath != "" {
		claudeProjectsPath = cfg.ClaudeProjectsPath
	}
	if projectQuery == "" && cfg != nil {
		projectQuery = fallback(cfg.ClaudeProjectPath, cfg.ClaudeProjectKey)
	}

	store, err := claude.NewStore(claudeProjectsPath)
	if err != nil {
		return nil, err
	}

	return store.LoadProjectByQuery(projectQuery, "")
}

func fallback(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
