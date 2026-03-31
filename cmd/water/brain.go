package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var brainCmd = &cobra.Command{
	Use:   "brain",
	Short: "Print an ASCII brain graph summary for the linked Claude project",
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshot, err := loadClaudeSnapshotForCommand(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("%s\n", snapshot.Project.Name)
		fmt.Printf("└─ project %s\n", snapshot.Project.Path)

		if len(snapshot.Memory) > 0 {
			fmt.Printf("   ├─ memo (%d)\n", len(snapshot.Memory))
			for _, memoryFile := range snapshot.Memory {
				fmt.Printf("   │  └─ %s\n", memoryFile.RelativePath)
			}
		}

		fmt.Printf("   ├─ sessions (%d)\n", len(snapshot.Sessions))
		for _, session := range snapshot.Sessions {
			fmt.Printf("   │  ├─ %s  %s\n", session.ID[:8], session.UpdatedAt.Format("2006-01-02"))
			fmt.Printf("   │  │  prompt: %s\n", fallback(session.PromptPreview, "(no prompt preview)"))
			if len(session.ToolNames) > 0 {
				fmt.Printf("   │  │  tools: %s\n", strings.Join(session.ToolNames, ", "))
			}
			if len(session.PathRefs) > 0 {
				fmt.Printf("   │  │  paths: %d\n", len(session.PathRefs))
			}
			for _, subagent := range session.Subagents {
				label := fallback(subagent.AgentType, subagent.ID)
				fmt.Printf("   │  │  subagent: %s\n", label)
			}
		}

		if len(snapshot.Tools) > 0 {
			fmt.Printf("   ├─ tools (%d)\n", len(snapshot.Tools))
			for _, tool := range snapshot.Tools[:min(8, len(snapshot.Tools))] {
				fmt.Printf("   │  └─ %s (%d)\n", tool.Name, tool.Count)
			}
		}

		if len(snapshot.Paths) > 0 {
			fmt.Printf("   └─ paths (%d)\n", len(snapshot.Paths))
			for _, pathItem := range snapshot.Paths[:min(10, len(snapshot.Paths))] {
				fmt.Printf("      └─ %s (%d)\n", pathItem.Path, pathItem.ReferenceCount)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(brainCmd)
	brainCmd.Flags().String("db-path", ".water", "Path to .water directory")
	brainCmd.Flags().String("project", "", "Claude project name, key, or path")
	brainCmd.Flags().String("claude-projects-path", "", "Path to ~/.claude/projects")
}
