# Skill: Go Cobra CLI

Panduan implementasi CLI Water menggunakan Cobra v2 + Viper.

---

## Setup Awal

```bash
go get github.com/spf13/cobra/v2
go get github.com/spf13/viper
cobra-cli init
cobra-cli add init
cobra-cli add serve
cobra-cli add watch
cobra-cli add export
cobra-cli add config
cobra-cli add install
```

---

## Struktur Root Command

```go
// cmd/water/root.go
package main

import (
    "os"
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "water",
    Short: "Visual brain of MCP agents",
    Long: `Water captures and visualizes what your Claude Code agent is thinking:
knowledge graphs, reasoning paths, and token flow.`,
    Version: "0.1.0",
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func main() {
    Execute()
}
```

---

## Pola Subcommand

Setiap subcommand mengikuti pola ini:

```go
// cmd/water/init.go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/water-viz/water/internal/config"
    "github.com/water-viz/water/internal/graph"
    "github.com/water-viz/water/internal/logger"
)

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize Water in current project",
    Long:  `Creates .water/ folder with DuckDB schema and config file.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Read flags
        dbPath, _ := cmd.Flags().GetString("db-path")
        port, _   := cmd.Flags().GetInt("port")
        mode, _   := cmd.Flags().GetString("embedding-mode")

        // 2. Business logic
        cfg := &config.Config{
            DBPath:        dbPath,
            Port:          port,
            EmbeddingMode: mode,
            Host:          "127.0.0.1",
            LogLevel:      "info",
            EnableWebSocket: true,
        }

        // 3. Create .water directory
        if err := graph.InitSchema(cmd.Context(), dbPath); err != nil {
            return fmt.Errorf("init schema: %w", err)
        }

        // 4. Save config
        cfgPath := config.GetConfigPath(dbPath)
        if err := cfg.Save(cfgPath); err != nil {
            return fmt.Errorf("save config: %w", err)
        }

        logger.Info("Water initialized", "path", dbPath, "port", port)
        fmt.Printf("✅ Water initialized at %s\n", dbPath)
        fmt.Printf("   Run: water serve\n")
        return nil
    },
}

func init() {
    rootCmd.AddCommand(initCmd)
    initCmd.Flags().String("db-path", ".water", "Path to .water directory")
    initCmd.Flags().String("embedding-mode", "local", "Embedding mode: local|api")
    initCmd.Flags().Int("port", 3141, "Web server port")
}
```

---

## Flag Patterns

### String flag dengan default
```go
cmd.Flags().String("db-path", ".water", "Path to .water directory")
val, _ := cmd.Flags().GetString("db-path")
```

### Bool flag (shorthand)
```go
cmd.Flags().BoolP("open-browser", "o", true, "Auto-open browser")
open, _ := cmd.Flags().GetBool("open-browser")
```

### Persistent flags (tersedia di semua subcommand)
```go
rootCmd.PersistentFlags().String("log-level", "info", "Log level: debug|info|warn|error")
```

### Required flag
```go
cmd.MarkFlagRequired("api-key")
```

---

## Error Handling di RunE

Selalu gunakan `RunE` (bukan `Run`) supaya error bisa di-propagate:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if err := doSomething(); err != nil {
        return fmt.Errorf("context: %w", err)  // wrap dengan %w
    }
    return nil
},
```

Cobra akan print error dan exit(1) otomatis.

---

## Config via Viper

```go
// internal/config/config.go
func LoadConfig(cfgPath string) (*Config, error) {
    v := viper.New()

    // Defaults
    v.SetDefault("port", 3141)
    v.SetDefault("host", "127.0.0.1")
    v.SetDefault("embedding_mode", "local")
    v.SetDefault("log_level", "info")
    v.SetDefault("enable_websocket", true)

    // File
    if cfgPath != "" {
        v.SetConfigFile(cfgPath)
        if err := v.ReadInConfig(); err != nil && !os.IsNotExist(err) {
            return nil, fmt.Errorf("read config: %w", err)
        }
    }

    // Environment overrides
    v.BindEnv("anthropic_api_key", "ANTHROPIC_API_KEY")
    v.BindEnv("port", "WATER_PORT")
    v.BindEnv("db_path", "WATER_DB_PATH")
    v.BindEnv("log_level", "WATER_LOG_LEVEL")

    return &Config{
        DBPath:          v.GetString("db_path"),
        Host:            v.GetString("host"),
        Port:            v.GetInt("port"),
        EmbeddingMode:   v.GetString("embedding_mode"),
        AnthropicAPIKey: v.GetString("anthropic_api_key"),
        LogLevel:        v.GetString("log_level"),
        EnableWebSocket: v.GetBool("enable_websocket"),
    }, nil
}
```

---

## Testing CLI Commands

```go
func TestInitCommand(t *testing.T) {
    tmpDir := t.TempDir()
    
    cmd := rootCmd
    cmd.SetArgs([]string{"init", "--db-path", tmpDir, "--port", "3142"})
    
    err := cmd.Execute()
    assert.NoError(t, err)
    
    // Verify .water folder created
    assert.DirExists(t, tmpDir)
    assert.FileExists(t, filepath.Join(tmpDir, "config.json"))
    assert.FileExists(t, filepath.Join(tmpDir, "database.duckdb"))
}
```

---

## Tips

- Gunakan `cmd.Context()` untuk pass context ke layer bawah — jangan buat context baru di handler
- Gunakan `cobra.OnInitialize()` jika ada setup global (misal: inisialisasi logger dari flag)
- Hindari global state — inject dependencies via closure atau struct
- `--help` otomatis di-generate Cobra; tulis `Short` dan `Long` yang informatif