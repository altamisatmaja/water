# Water Skills

Skill files berisi pola, konvensi, dan contoh kode yang bisa digunakan kembali oleh Claude Code agent saat mengerjakan proyek Water.

## Daftar Skills

| File | Topik |
|------|-------|
| `go-cobra-cli.md` | Pola CLI dengan Cobra + Viper |
| `duckdb-go.md` | DuckDB Go driver: koneksi, query, schema |
| `svelte-cytoscape.md` | Graph visualization dengan Cytoscape.js di Svelte |
| `event-capture.md` | Event schema, JSONL write/read, streaming |
| `cross-compile.md` | Multi-platform build + GitHub Actions release |

## Cara Pakai

Setiap skill file bersifat **self-contained**. Agent bisa membaca satu skill secara langsung untuk mendapatkan context yang relevan sebelum menghasilkan kode.

Contoh penggunaan di dalam prompt agent:
```
Read skills/duckdb-go.md, then implement internal/graph/client.go
```