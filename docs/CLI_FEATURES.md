# CLI Features Guide

This guide covers all the CLI features available in unqueryvet.

## Table of Contents

- [Version Information](#version-information)
- [Output Modes](#output-modes)
- [Colored Output](#colored-output)
- [Analysis Options](#analysis-options)
- [Interactive Fix Mode](#interactive-fix-mode)
- [Configuration File](#configuration-file)
- [Enhanced Error Messages](#enhanced-error-messages)

---

## Version Information

Display detailed version information including build details:

```bash
$ unqueryvet -version

unqueryvet version 1.5.0-dev
  commit: 108c957
  built:  2026-01-11T08:42:12Z
  by:     unknown
  go:     go1.25.5
  platform: windows/amd64
```

**Use Cases:**
- Verify which version is installed
- Check build information for debugging
- Ensure you're running the latest version in CI/CD

---

## Output Modes

### Verbose Mode

Enable detailed output with explanations, examples, and performance impact:

```bash
$ unqueryvet -verbose ./myapp.go
```

**Features:**
- Shows debug messages
- Displays detailed diagnostic information
- Includes examples of how to fix issues
- Shows performance impact of problems
- Links to documentation

**Example Output:**
```
DEBUG: unqueryvet v1.5.0-dev
myapp.go:42:11: avoid SELECT * - explicitly specify needed columns

Using SELECT * can lead to unexpected behavior when schema changes
and wastes network bandwidth by transferring unnecessary data.

Example fix:
  - query := "SELECT * FROM users"
  + query := "SELECT id, name, email FROM users"

Impact: 30-70% performance improvement for tables with many columns.
```

### Quiet Mode

Suppress warnings and show only errors:

```bash
$ unqueryvet -quiet ./...
```

**Use Cases:**
- CI/CD pipelines where you only care about errors
- Quick checks during development
- Reducing noise in large codebases

**Example:**
```bash
# Normal mode
$ unqueryvet ./...
file1.go:10: warning: avoid SELECT *
file2.go:20: warning: avoid SELECT *
file3.go:30: error: invalid SQL syntax

# Quiet mode (only errors)
$ unqueryvet -quiet ./...
file3.go:30: error: invalid SQL syntax
```

---

## Colored Output

Automatic colored output for better readability:

- **Red**: Errors
- **Yellow**: Warnings
- **Green**: Success messages
- **Gray**: Debug information
- **Bold**: File locations and important info

### Disabling Colors

```bash
# Disable via flag
$ unqueryvet -no-color ./...

# Disable via environment variable
$ NO_COLOR=1 unqueryvet ./...
```

**Auto-Detection:**
The tool automatically detects if output is to a terminal and disables colors for:
- Pipes and redirects
- Non-terminal output (files, CI/CD logs)
- Dumb terminals (`TERM=dumb`)

---

## Analysis Options

### N+1 Query Detection

Enable detection of potential N+1 query problems:

```bash
$ unqueryvet -n1 ./...
```

Detects queries inside loops that could cause performance issues.

### SQL Injection Detection

Enable detection of potential SQL injection vulnerabilities:

```bash
$ unqueryvet -sqli ./...
```

Detects:
- `fmt.Sprintf` with SQL queries
- String concatenation in SQL
- Direct variable interpolation

### Statistics

Show analysis statistics after completion:

```bash
$ unqueryvet -stats ./...
```

Displays:
- Number of files analyzed
- Number of packages loaded
- Issues by type breakdown
- Analysis duration

---

## Interactive Fix Mode

Fix issues interactively with a terminal UI:

```bash
$ unqueryvet -fix ./...
```

### Controls

| Category | Key | Action |
|----------|-----|--------|
| **Navigation** | `↑/k` | Previous issue |
| | `↓/j` | Next issue |
| | `g` | Go to first issue |
| | `G` | Go to last issue |
| **Actions** | `Enter/a` | Apply fix |
| | `s` | Skip issue |
| | `u` | Undo last action |
| | `p` | Toggle preview |
| **Batch** | `A` | Apply all remaining |
| | `S` | Skip all remaining |
| | `R` | Reset all actions |
| **Other** | `e` | Export results to JSON |
| | `?` | Toggle help |
| | `q/Esc` | Quit |

### Example Session

```
Found 15 issues. Review each one:

[1/15] internal/api/users.go:42:15
─────────────────────────────────────
  41 | func getUsers(db *sql.DB) {
  42 |     query := "SELECT * FROM users"
     |              ^^^^^^^^^^^^^^^^^^^^^ avoid SELECT *
  43 |     rows, _ := db.Query(query)

Suggestions:
  1. SELECT id, username, email, created_at (from struct User)
  2. SELECT id, username, email
  3. Skip this issue
  4. Edit manually

Your choice [1-4]: _
```

---

## Configuration File

Create a `.unqueryvet.yaml` file in your project root for persistent configuration:

### Basic Configuration

```yaml
# .unqueryvet.yaml
severity: warning
check-sql-builders: true
check-aliased-wildcard: true
```

### Full Example

```yaml
# Diagnostic severity: "error" or "warning"
severity: warning

# Core analysis options
check-sql-builders: true
check-aliased-wildcard: true
check-string-concat: true
check-format-strings: true
check-string-builder: true
check-subqueries: true

# Advanced analysis
check-n1-queries: true
check-sql-injection: true

# File patterns to ignore (glob)
ignored-files:
  - "*_test.go"
  - "testdata/**"
  - "vendor/**"

# Allowed SELECT * patterns (regex)
allowed-patterns:
  - "SELECT \\* FROM temp_.*"
```

### Configuration Precedence

1. Command-line flags (highest priority)
2. `.unqueryvet.yaml` in current or parent directories
3. Default settings (lowest priority)

### Auto-Discovery

The tool searches for configuration files in this order:
1. Current directory: `./.unqueryvet.yaml` or `./.unqueryvet.yml`
2. Parent directories (up to root)
3. Uses default settings if no config found

### Common Configurations

#### Strict Mode (Fail on Any Issue)

```yaml
severity: error
check-sql-builders: true
check-aliased-wildcard: true
check-string-concat: true
check-format-strings: true
check-string-builder: true
check-subqueries: true
```

#### Permissive Mode (Only Basic Checks)

```yaml
severity: warning
check-sql-builders: false
check-aliased-wildcard: false
check-string-concat: false
```

#### Ignore Test Files

```yaml
ignored-files:
  - "*_test.go"
  - "testdata/**"
  - "mocks/**"
```

---

## Enhanced Error Messages

Error messages include context and suggestions:

### Normal Mode
```
file.go:10: avoid SELECT * - explicitly specify needed columns for better performance, maintainability and stability
```

### Verbose Mode
```
file.go:10: avoid SELECT * - explicitly specify needed columns

Using SELECT * can lead to unexpected behavior when schema changes
and wastes network bandwidth by transferring unnecessary data.

Example fix:
  - query := "SELECT * FROM users"
  + query := "SELECT id, name, email FROM users"

Specify only the columns you actually need in your application.

Impact: 30-70% performance improvement for tables with many columns.
        Prevents breaking changes when schema is modified.
```

---

## Usage Examples

### Development Workflow

```bash
# Quick check during development (normal output)
$ unqueryvet ./...

# Detailed check before commit (verbose)
$ unqueryvet -verbose ./...

# Fast check (quiet mode)
$ unqueryvet -quiet ./...

# Full analysis with all checks
$ unqueryvet -n1 -sqli -stats ./...

# Interactive fix mode
$ unqueryvet -fix ./...
```

### CI/CD Integration

```yaml
# GitHub Actions
name: SQL Lint
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
      - name: Install unqueryvet
        run: go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet@latest
      
      - name: Run unqueryvet
        run: unqueryvet -quiet -n1 -sqli ./...
```

### Git Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running unqueryvet..."
if ! unqueryvet -quiet ./...; then
    echo "unqueryvet found issues. Commit aborted."
    echo "Run 'unqueryvet -verbose ./...' for details."
    exit 1
fi

echo "unqueryvet passed"
```

### Docker

```dockerfile
FROM golang:1.24-alpine

# Install unqueryvet
RUN go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet@latest

# Copy config
COPY .unqueryvet.yaml /workspace/.unqueryvet.yaml

# Run checks
WORKDIR /workspace
CMD ["unqueryvet", "-quiet", "./..."]
```

---

## CLI Flags Reference

| Flag | Description |
|------|-------------|
| `-version` | Print version information |
| `-verbose` | Enable verbose output with detailed explanations |
| `-quiet` | Quiet mode (only errors) |
| `-stats` | Show analysis statistics |
| `-no-color` | Disable colored output |
| `-n1` | Detect potential N+1 query problems |
| `-sqli` | Detect potential SQL injection vulnerabilities |
| `-fix` | Interactive fix mode |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Warnings found |
| 2 | Errors found |
| 3 | Analysis failed |

---

## Troubleshooting

### Colors Not Working

**Problem:** Colors not showing in terminal

**Solutions:**
1. Check terminal supports colors: `echo $TERM`
2. Ensure NO_COLOR is not set: `echo $NO_COLOR`
3. Try forcing colors: `TERM=xterm-256color unqueryvet ./...`

### Config Not Loading

**Problem:** `.unqueryvet.yaml` not being read

**Debug:**
```bash
# Check config file location
$ find . -name ".unqueryvet.yaml"

# Run with verbose to see what config is loaded
$ unqueryvet -verbose ./... 2>&1 | grep -i config

# Validate YAML syntax
$ cat .unqueryvet.yaml | yaml-lint
```

### High Memory Usage

**Problem:** Analyzer using too much memory on large codebases

**Solutions:**
1. Analyze packages incrementally:
   ```bash
   for pkg in $(go list ./...); do
     unqueryvet "$pkg"
   done
   ```

2. Ignore large generated files:
   ```yaml
   # .unqueryvet.yaml
   ignored-files:
     - "**/*_generated.go"
     - "**/wire_gen.go"
   ```

---

## See Also

- [Custom Rules DSL](DSL.md)
- [Main README](../README.md)
