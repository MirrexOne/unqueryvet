// Command unqueryvet-lsp runs the Language Server Protocol server for unqueryvet.
// It provides real-time SQL analysis in editors that support LSP.
//
// Usage:
//
//	unqueryvet-lsp [flags]
//
// The server communicates via stdin/stdout using JSON-RPC 2.0.
//
// Flags:
//
//	-version    Print version information and exit
//	-help       Print help message and exit
//
// VS Code Configuration:
//
// Add to settings.json:
//
//	{
//	  "go.lintTool": "unqueryvet-lsp"
//	}
//
// Or use with a generic LSP client extension.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/MirrexOne/unqueryvet/internal/lsp"
	"github.com/MirrexOne/unqueryvet/internal/version"
)

var (
	versionFlag = flag.Bool("version", false, "print version information and exit")
	helpFlag    = flag.Bool("help", false, "print help message and exit")
	stdioFlag   = flag.Bool("stdio", false, "use stdio for communication (default, for compatibility)")
)

func main() {
	flag.Parse()

	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	if *versionFlag {
		info := version.GetInfo()
		fmt.Printf("unqueryvet-lsp %s\n", info.Version)
		os.Exit(0)
	}

	// Create a context that cancels on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Create and run the LSP server
	server := lsp.NewStdioServer()
	if err := server.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`unqueryvet-lsp - Language Server Protocol server for SQL analysis

USAGE:
  unqueryvet-lsp [flags]

DESCRIPTION:
  unqueryvet-lsp provides real-time SQL analysis in editors that support
  the Language Server Protocol (LSP). It detects SELECT * usage and
  provides diagnostics, code actions, and hover information.

FEATURES:
  • Real-time diagnostics for SELECT * in SQL queries
  • Quick fixes to replace SELECT * with explicit columns
  • Hover information explaining why SELECT * is problematic
  • Column name completions for SQL strings

FLAGS:
  -version    Print version information and exit
  -help       Print this help message and exit

EDITOR SETUP:

  VS Code:
    Install a generic LSP client extension and configure:
    {
      "languageServerSettings": {
        "go": {
          "command": "unqueryvet-lsp"
        }
      }
    }

  Neovim (with nvim-lspconfig):
    require'lspconfig'.unqueryvet.setup{}

  Vim (with vim-lsp):
    autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'unqueryvet',
      \ 'cmd': {server_info->['unqueryvet-lsp']},
      \ 'whitelist': ['go'],
      \ })

EXAMPLES:
  # Start the LSP server (typically called by editor)
  unqueryvet-lsp

  # Check version
  unqueryvet-lsp -version

DOCUMENTATION:
  https://github.com/MirrexOne/unqueryvet`)
}
