# Verified Features - unqueryvet v1.5.0

This document lists all implemented and verified features in unqueryvet.

## Core Analysis Features

### SELECT * Detection
- [x] Basic SELECT * in database/sql queries
- [x] Aliased wildcard detection (SELECT t.* FROM table t)
- [x] String concatenation detection ("SELECT * " + "FROM table")
- [x] Format string detection (fmt.Sprintf("SELECT * FROM %s", table))
- [x] strings.Builder detection
- [x] Subquery detection (SELECT * FROM (SELECT * FROM ...))

### N+1 Query Detection
- [x] Detection of queries in for/range loops
- [x] Support for multiple ORM methods (Query, QueryRow, Exec, Find, First, etc.)
- [x] Indirect N+1 detection via function calls
- [x] Nested transaction detection in loops
- [x] Suggestions for JOIN/IN optimization

### SQL Injection Detection
- [x] Taint tracking for user input
- [x] HTTP request parameter detection (r.URL.Query, r.FormValue, etc.)
- [x] os.Args and environment variable tracking
- [x] Severity levels: CRITICAL, HIGH, MEDIUM, LOW
- [x] Safe parameterized query detection (skips false positives)
- [x] Detailed fix suggestions

## SQL Builder Support

### Fully Supported (12 builders)
- [x] Squirrel (github.com/Masterminds/squirrel)
- [x] GORM (gorm.io/gorm)
- [x] SQLx (github.com/jmoiron/sqlx)
- [x] Ent (entgo.io/ent)
- [x] PGX (github.com/jackc/pgx)
- [x] Bun (github.com/uptrace/bun)
- [x] SQLBoiler (github.com/volatiletech/sqlboiler)
- [x] Jet (github.com/go-jet/jet)
- [x] sqlc (generated code detection)
- [x] goqu (github.com/doug-martin/goqu)
- [x] rel (github.com/go-rel/rel)
- [x] reform (gopkg.in/reform.v1)

## Interactive Features

### Interactive TUI (Bubble Tea)
- [x] Issue navigation (j/k, up/down)
- [x] Diff preview (before/after)
- [x] Apply fix (a key)
- [x] Skip issue (s key)
- [x] Batch operations (A - apply all, S - skip all)
- [x] Undo support (u key)
- [x] Export to JSON (e key)
- [x] Smart fix suggestions based on issue type
- [x] Status bar with progress

### Watch Mode
- [x] File system watching (fsnotify)
- [x] Config file reload on change
- [x] Statistics output

### Fix Mode
- [x] Automatic code fixes
- [x] Backup before changes
- [x] Dry-run mode
- [x] Verbose output

## IDE Integration

### LSP Server
- [x] textDocument/didOpen
- [x] textDocument/didChange
- [x] textDocument/didSave
- [x] textDocument/didClose
- [x] textDocument/publishDiagnostics
- [x] textDocument/codeAction
- [x] WebSocket API support
- [x] stdio transport

### VS Code Extension
- [x] LSP client integration
- [x] Status bar with issue count
- [x] Commands: Analyze File, Analyze Workspace, Fix All
- [x] Settings integration
- [x] Diagnostic highlighting

## Output Formats

- [x] Text (default, with colors)
- [x] JSON
- [x] SARIF (for CI/CD)

## Configuration

### Config File (.unqueryvet.yaml)
- [x] check-sql-builders
- [x] allowed-patterns (regex)
- [x] ignored-functions
- [x] ignored-files (glob)
- [x] severity (error/warning)
- [x] check-aliased-wildcard
- [x] check-string-concat
- [x] check-format-strings
- [x] check-string-builder
- [x] check-subqueries
- [x] sql-builders (per-builder enable/disable)

### CLI Flags
- [x] -version
- [x] -verbose
- [x] -quiet
- [x] -stats
- [x] -no-color
- [x] -n1 (N+1 detection)
- [x] -sqli (SQL injection detection)
- [x] -fix (interactive TUI mode)

### Planned CLI Flags (not yet implemented)
- [ ] -config (custom config path)
- [ ] -format (text/json/sarif output format)
- [ ] -watch (file watching mode)
- [ ] -severity (override severity level)

## CI/CD Integration

- [x] golangci-lint plugin
- [x] GitHub Actions example
- [x] GitLab CI example
- [x] Exit codes (0 = no issues, 1 = issues found, 2 = error)
