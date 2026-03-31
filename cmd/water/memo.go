package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var memoCmd = &cobra.Command{
	Use:   "memo",
	Short: "Show Claude memory files for the linked project",
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshot, err := loadClaudeSnapshotForCommand(cmd)
		if err != nil {
			return err
		}

		if len(snapshot.Memory) == 0 {
			fmt.Printf("%s\nNo Claude memory files found in %s\n", snapshot.Project.Name, snapshot.Project.Path)
			return nil
		}

		fmt.Printf("%s\n", snapshot.Project.Name)
		fmt.Printf("Memory files: %d\n\n", len(snapshot.Memory))
		for i, memoryFile := range snapshot.Memory {
			fmt.Printf("%d. %s\n", i+1, memoryFile.RelativePath)
			fmt.Printf("   %s\n\n", memoryFile.Preview)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(memoCmd)
	memoCmd.Flags().String("db-path", ".water", "Path to .water directory")
	memoCmd.Flags().String("project", "", "Claude project name, key, or path")
	memoCmd.Flags().String("claude-projects-path", "", "Path to ~/.claude/projects")
}
