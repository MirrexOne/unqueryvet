# Changelog

All notable changes to the Unqueryvet VS Code extension will be documented in this file.

## [1.0.0] - 2025-01-14

### Added

- Initial release
- SELECT * detection in raw SQL and SQL builders
- N+1 query detection (queries inside loops)
- SQL injection vulnerability scanning
- Support for 12 SQL builders:
  - Squirrel, GORM, SQLx, Ent, PGX, Bun
  - SQLBoiler, Jet, sqlc, goqu, rel, reform
- Real-time diagnostics via LSP
- Quick fix suggestions
- Workspace-wide analysis
- Configuration via `.unqueryvet.yaml`
- Status bar with issue count
- Commands: Analyze File, Analyze Workspace, Fix All, Restart Server
