# Changelog

All notable changes to the Unqueryvet VS Code extension will be documented in this file.

## [1.5.3] - 2025-01-21

### Added

- **Transaction leak detection (tx-leak)** - Detects unclosed SQL transactions
- **17 violation types** - Comprehensive detection patterns for transaction issues
- **Support for multiple Begin methods** - database/sql, sqlx, pgx, bun, ent

### Changed

- Improved README documentation
- Updated GitHub Actions example

## [1.5.1] - 2025-01-19

### Added

- **Automatic LSP server download** - Extension now automatically downloads and installs the LSP server on first use
- **Cross-platform support** - Pre-built binaries for Windows, Linux, macOS (amd64 and arm64)
- **Intelligent LSP discovery** - Automatically finds LSP in PATH, GOPATH/bin, or downloads if missing
- **Progress indicators** - Download progress bar with size information
- **Multiple installation options** - Automatic download, manual installation, or custom path

### Changed

- **Simplified installation** - No need to manually install LSP server via `go install`
- **Improved error messages** - Better guidance when LSP server is not found
- **Updated README** - Added automatic installation documentation
- **Publisher ID** - Changed to `mirrexdev` for consistency

### Fixed

- Fixed extension ID in restart command
- Improved context handling for extension lifecycle

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
