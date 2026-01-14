# Unqueryvet for VS Code

A comprehensive SQL analysis extension for Go that detects `SELECT *` usage, N+1 query problems, and SQL injection vulnerabilities.

## Features

- **SELECT * Detection** - Finds `SELECT *` in raw SQL, SQL builders, and templates
- **N+1 Query Detection** - Identifies queries inside loops
- **SQL Injection Scanner** - Detects `fmt.Sprintf` and string concatenation vulnerabilities
- **12 SQL Builder Support** - Squirrel, GORM, SQLx, Ent, PGX, Bun, SQLBoiler, Jet, sqlc, goqu, rel, reform
- **Real-time Diagnostics** - See issues as you type
- **Quick Fixes** - One-click fixes for detected issues

## Requirements

- VS Code 1.85.0 or higher
- Go installed
- `unqueryvet-lsp` binary installed:

```bash
go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet-lsp@latest
```

## Installation

1. Install the extension from VS Code Marketplace
2. Install the LSP server: `go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet-lsp@latest`
3. Open a Go file - the extension will activate automatically

## Extension Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `unqueryvet.enable` | Enable/disable the extension | `true` |
| `unqueryvet.lspPath` | Custom path to unqueryvet-lsp binary | Auto-detected |
| `unqueryvet.analyzeOnSave` | Analyze files when saved | `true` |
| `unqueryvet.analyzeOnType` | Real-time analysis while typing | `true` |
| `unqueryvet.severity` | Diagnostic severity level | `warning` |
| `unqueryvet.checkN1Queries` | Enable N+1 query detection | `true` |
| `unqueryvet.checkSQLInjection` | Enable SQL injection detection | `true` |

## Commands

| Command | Description |
|---------|-------------|
| `Unqueryvet: Analyze File` | Analyze the current Go file |
| `Unqueryvet: Analyze Workspace` | Analyze all Go files in workspace |
| `Unqueryvet: Fix All` | Apply all available fixes |
| `Unqueryvet: Show Output` | Show extension output channel |
| `Unqueryvet: Restart Server` | Restart the language server |

## Detection Examples

### SELECT * Detection

```go
// Detected as issue
query := "SELECT * FROM users"

// Good - explicit columns
query := "SELECT id, name, email FROM users"
```

### N+1 Query Detection

```go
// Detected as issue - query inside loop
for _, user := range users {
    orders, _ := db.Query("SELECT * FROM orders WHERE user_id = ?", user.ID)
}

// Good - use JOIN or batch query
query := "SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id"
```

### SQL Injection Detection

```go
// Detected as issue
query := fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", userName)

// Good - parameterized query
query := "SELECT * FROM users WHERE name = ?"
db.Query(query, userName)
```

## Configuration File

Create `.unqueryvet.yaml` in your project root for advanced configuration:

```yaml
rules:
  select-star: warning
  n1-queries: warning
  sql-injection: error

ignore:
  - "*_test.go"
  - "vendor/**"

allow:
  - "COUNT(*)"
```

## Troubleshooting

### LSP Server Not Found

If you see "unqueryvet-lsp not found", ensure:

1. The binary is installed: `go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet-lsp@latest`
2. `$GOPATH/bin` is in your PATH
3. Or set custom path: `"unqueryvet.lspPath": "/path/to/unqueryvet-lsp"`

### No Diagnostics Appearing

1. Ensure the file is a `.go` file
2. Check Output panel (View → Output → Unqueryvet)
3. Try restarting the server: `Unqueryvet: Restart Server`

## Links

- [GitHub Repository](https://github.com/MirrexOne/unqueryvet)
- [Documentation](https://github.com/MirrexOne/unqueryvet/blob/main/README.md)
- [Report Issues](https://github.com/MirrexOne/unqueryvet/issues)

## License

MIT License - see [LICENSE](https://github.com/MirrexOne/unqueryvet/blob/main/LICENSE) for details.
