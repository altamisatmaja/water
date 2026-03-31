package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Build Water from this clone and install a user-level command shim",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := detectRepoRoot()
		if err != nil {
			return err
		}

		binaryPath, err := buildLocalWaterBinary(repoRoot)
		if err != nil {
			return err
		}

		linkPath, binDir, err := installUserCommand(binaryPath)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Water install complete\n\n")
		fmt.Printf("Repository:\n  %s\n\n", repoRoot)
		fmt.Printf("Built binary:\n  %s\n\n", binaryPath)
		fmt.Printf("Command shim:\n  %s\n\n", linkPath)
		if pathContains(binDir) {
			fmt.Printf("PATH:\n  %s is already on PATH\n\n", binDir)
		} else {
			fmt.Printf("PATH:\n")
			fmt.Printf("  Add this directory to PATH:\n")
			fmt.Printf("    %s\n\n", binDir)
			fmt.Printf("  Run this command in your terminal:\n")
			fmt.Printf("    %s\n\n", shellPathHint(binDir))
			fmt.Printf("  Then open a new terminal and run:\n")
			fmt.Printf("    water --help\n\n")
		}
		fmt.Printf("Next:\n")
		fmt.Printf("  water --help\n")
		fmt.Printf("  water serve\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func detectRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}

	for dir := cwd; ; dir = filepath.Dir(dir) {
		if isWaterRepo(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return "", errors.New("water install must be run from the cloned Water repository (or one of its subdirectories)")
}

func isWaterRepo(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "cmd", "water", "main.go")); err != nil {
		return false
	}
	return true
}

func buildLocalWaterBinary(repoRoot string) (string, error) {
	binDir := filepath.Join(repoRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", binDir, err)
	}

	binaryName := "water"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(binDir, binaryName)

	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/water")
	buildCmd.Dir = repoRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return "", fmt.Errorf("build water: %w", err)
	}

	return binaryPath, nil
}

func installUserCommand(binaryPath string) (linkPath, binDir string, err error) {
	binDir, err = userBinDir()
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", "", fmt.Errorf("mkdir %s: %w", binDir, err)
	}

	if runtime.GOOS == "windows" {
		linkPath = filepath.Join(binDir, "water.cmd")
		content := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", binaryPath)
		if err := os.WriteFile(linkPath, []byte(content), 0o755); err != nil {
			return "", "", fmt.Errorf("write %s: %w", linkPath, err)
		}
		return linkPath, binDir, nil
	}

	linkPath = filepath.Join(binDir, "water")
	_ = os.Remove(linkPath)
	if err := os.Symlink(binaryPath, linkPath); err != nil {
		return "", "", fmt.Errorf("symlink %s -> %s: %w", linkPath, binaryPath, err)
	}
	return linkPath, binDir, nil
}

func userBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "bin"), nil
	case "darwin":
		return filepath.Join(home, ".local", "bin"), nil
	default:
		return filepath.Join(home, ".local", "bin"), nil
	}
}

func pathContains(dir string) bool {
	target := filepath.Clean(dir)
	for _, item := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(item) == target {
			return true
		}
	}
	return false
}

func shellPathHint(binDir string) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf(`[Environment]::SetEnvironmentVariable("Path", $env:Path + ";%s", "User")`, binDir)
	case "darwin":
		return fmt.Sprintf(`echo 'export PATH="%s:$PATH"' >> ~/.zshrc`, binDir)
	default:
		shell := strings.ToLower(filepath.Base(os.Getenv("SHELL")))
		if shell == "fish" {
			return fmt.Sprintf(`fish_add_path %s`, binDir)
		}
		return fmt.Sprintf(`echo 'export PATH="%s:$PATH"' >> ~/.bashrc`, binDir)
	}
}
