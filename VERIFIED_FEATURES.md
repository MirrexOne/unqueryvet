# Verified Features - unqueryvet

This document lists all implemented and verified features in unqueryvet.

> **Note**: All three detection rules (SELECT *, N+1, SQL Injection) are now **enabled by default**.

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

### Fix Mode (`-fix` flag)
- [x] Interactive issue review
- [x] Step-through fix application
- [x] Skip individual issues

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

## Configuration

### Config File (.unqueryvet.yaml)
- [x] rules (select-star, n1-queries, sql-injection with severity levels)
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
- [x] custom-rules (DSL-based custom rules)

### CLI Flags
- [x] -version
- [x] -verbose
- [x] -quiet
- [x] -stats
- [x] -no-color
- [x] -n1 (force enable N+1 detection, overrides config)
- [x] -sqli (force enable SQL injection detection, overrides config)
- [x] -fix (interactive TUI mode)

> **Note**: `-n1` and `-sqli` flags now act as overrides. All rules are enabled by default via config.

## CI/CD Integration

- [x] golangci-lint plugin
- [x] GitHub Actions example
- [x] GitLab CI example
- [x] Exit codes (0 = no issues, 1 = warnings, 2 = errors, 3 = analysis failed)
