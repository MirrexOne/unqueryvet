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
- No other dependencies required! The LSP server will be downloaded automatically.

## Installation

1. Install the extension from VS Code Marketplace
2. Open a Go file - the extension will detect if LSP server is missing
3. Click "Download" when prompted to automatically install the LSP server
4. That's it! The extension is ready to use

### Manual Installation (Optional)

If you prefer to install the LSP server manually:

```bash
go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet-lsp@latest
```

The extension will automatically detect manually installed LSP servers in:
- System PATH
- `$GOPATH/bin`
- Custom path specified in settings

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

The extension will automatically prompt you to download the LSP server if it's not found. If you encounter issues:

1. **Allow automatic download**: Click "Download" when prompted
2. **Manual installation**: Run `go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet-lsp@latest`
3. **Custom path**: Set `"unqueryvet.lspPath"` in settings if installed in a non-standard location
4. **Check logs**: View → Output → Select "Unqueryvet" to see detailed error messages

### Download Fails

If automatic download fails:

1. Check your internet connection
2. Verify you can access GitHub
3. Try manual installation: `go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet-lsp@latest`
4. Or download manually from [GitHub Releases](https://github.com/MirrexOne/unqueryvet/releases)

### No Diagnostics Appearing

1. Ensure the file is a `.go` file
2. Check Output panel (View → Output → Unqueryvet)
3. Verify LSP server is running (check status bar)
4. Try restarting the server: `Unqueryvet: Restart Server`

## Links

- [GitHub Repository](https://github.com/MirrexOne/unqueryvet)
- [Documentation](https://github.com/MirrexOne/unqueryvet/blob/main/README.md)
- [Report Issues](https://github.com/MirrexOne/unqueryvet/issues)

## License

MIT License - see [LICENSE](https://github.com/MirrexOne/unqueryvet/blob/main/LICENSE) for details.
