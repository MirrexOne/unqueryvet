# IDE Integration Guide

This guide covers how to integrate unqueryvet with various IDEs and code editors for real-time SQL analysis.

## Table of Contents

- [Overview](#overview)
- [GoLand / IntelliJ IDEA](#goland--intellij-idea)
- [VS Code](#vs-code)
- [Vim / Neovim](#vim--neovim)
- [Other LSP-Compatible Editors](#other-lsp-compatible-editors)
- [Troubleshooting](#troubleshooting)

---

## Overview

unqueryvet provides IDE integration through two mechanisms:

1. **LSP Server** (`unqueryvet-lsp`) - Language Server Protocol for real-time diagnostics
2. **Native Plugins** - GoLand and VS Code extensions with additional features

> **Note**: All three detection rules (SELECT *, N+1, SQL Injection) are **enabled by default**. No additional configuration is required.

### Installing the LSP Server

```bash
go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet-lsp@latest
```

Verify installation:
```bash
unqueryvet-lsp -version
```

---

## GoLand / IntelliJ IDEA

### Option 1: Install from JetBrains Marketplace

1. Open Settings → Plugins → Marketplace
2. Search for "unqueryvet"
3. Click Install and restart IDE

### Option 2: Build from Source

```bash
cd extensions/goland
./gradlew buildPlugin
```

The plugin will be built to `build/distributions/unqueryvet-*.zip`

Install:
1. Settings → Plugins → ⚙️ → Install Plugin from Disk
2. Select the built `.zip` file
3. Restart IDE

### Configuration

Navigate to **Settings → Tools → unqueryvet**

| Setting | Description | Default |
|---------|-------------|---------|
| **Enabled** | Enable/disable the plugin | `true` |
| **Binary Path** | Path to unqueryvet binary | Auto-detected |
| **Enable N+1 Detection** | Detect N+1 query problems | `false` |
| **Enable SQL Injection Detection** | Scan for SQL injection | `false` |
| **Auto Fix** | Automatically apply fixes | `false` |
| **Exclude Patterns** | Files/directories to exclude | `*_test.go, testdata/**` |

### Features

#### Real-time Diagnostics
- Warnings appear as you type
- Highlighted directly in the editor
- Hover for detailed explanations

#### Quick Fixes
- Press `Alt+Enter` on a warning
- Select "Replace SELECT * with explicit columns"
- Columns are suggested based on context

#### Tool Window
- View → Tool Windows → unqueryvet
- Shows all issues in the project
- Click to navigate to source

#### Actions
- **Run Analysis** - Analyze current file or project
- **Fix All Issues** - Apply all suggested fixes

### Keyboard Shortcuts

| Action | Shortcut |
|--------|----------|
| Run Analysis | `Ctrl+Alt+U` (customizable) |
| Quick Fix | `Alt+Enter` |
| Navigate to Issue | Click in Tool Window |

---

## VS Code

### Option 1: Install Extension

The VS Code extension is available in `extensions/vscode/`.

Build and install:
```bash
cd extensions/vscode
npm install
npm run compile
npm run package
# Install the generated .vsix file
```

### Option 2: Manual LSP Configuration

Add to your `.vscode/settings.json`:

```json
{
  "unqueryvet.enable": true,
  "unqueryvet.path": "unqueryvet-lsp",
  "unqueryvet.args": [],
  "unqueryvet.trace.server": "off"
}
```

All rules (SELECT *, N+1, SQL Injection) are enabled by default - no additional args needed.

For verbose logging:
```json
{
  "unqueryvet.enable": true,
  "unqueryvet.path": "unqueryvet-lsp",
  "unqueryvet.trace.server": "verbose"
}
```

### Features

#### Diagnostics
- Real-time warnings in Problems panel
- Inline squiggles on problematic code
- Hover tooltips with explanations

#### Status Bar
- Shows issue count in status bar
- Click to open Problems panel

#### Commands

Open Command Palette (`Ctrl+Shift+P`) and type:

| Command | Description |
|---------|-------------|
| `unqueryvet: Analyze File` | Analyze current file |
| `unqueryvet: Analyze Workspace` | Analyze entire workspace |
| `unqueryvet: Fix All` | Apply all fixes in current file |
| `unqueryvet: Show Output` | Show LSP server logs |

### Recommended Extensions

For the best experience, also install:
- [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go) - Go language support
- [Error Lens](https://marketplace.visualstudio.com/items?itemName=usernamehw.errorlens) - Inline error display

---

## Vim / Neovim

### Neovim with nvim-lspconfig

Add to your `init.lua`:

```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

-- Define unqueryvet LSP
if not configs.unqueryvet then
  configs.unqueryvet = {
    default_config = {
      cmd = { 'unqueryvet-lsp' },
      filetypes = { 'go' },
      root_dir = lspconfig.util.root_pattern('go.mod', '.git'),
      settings = {},
    },
  }
end

-- Setup
lspconfig.unqueryvet.setup({
  on_attach = function(client, bufnr)
    -- Your on_attach configuration
  end,
})
```

All rules are enabled by default - no additional flags needed.

### Vim with vim-lsp

Add to your `.vimrc`:

```vim
if executable('unqueryvet-lsp')
  au User lsp_setup call lsp#register_server({
    \ 'name': 'unqueryvet',
    \ 'cmd': {server_info->['unqueryvet-lsp']},
    \ 'allowlist': ['go'],
    \ })
endif
```

### CoC.nvim

Add to `coc-settings.json`:

```json
{
  "languageserver": {
    "unqueryvet": {
      "command": "unqueryvet-lsp",
      "filetypes": ["go"],
      "rootPatterns": ["go.mod", ".git"]
    }
  }
}
```

All rules are enabled by default.

---

## Other LSP-Compatible Editors

### Sublime Text (LSP package)

Add to LSP settings:

```json
{
  "clients": {
    "unqueryvet": {
      "enabled": true,
      "command": ["unqueryvet-lsp"],
      "selector": "source.go"
    }
  }
}
```

### Emacs (lsp-mode)

Add to your config:

```elisp
(lsp-register-client
 (make-lsp-client
  :new-connection (lsp-stdio-connection '("unqueryvet-lsp"))
  :major-modes '(go-mode)
  :server-id 'unqueryvet))
```

### Helix

Add to `languages.toml`:

```toml
[[language]]
name = "go"
language-servers = ["gopls", "unqueryvet"]

[language-server.unqueryvet]
command = "unqueryvet-lsp"
```

All rules are enabled by default.

---

## Troubleshooting

### LSP Server Not Starting

1. **Check installation:**
   ```bash
   which unqueryvet-lsp
   unqueryvet-lsp -version
   ```

2. **Check PATH:**
   Ensure `$GOPATH/bin` is in your PATH:
   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

3. **Check logs:**
   - VS Code: View → Output → unqueryvet
   - Neovim: `:LspLog`
   - GoLand: Help → Show Log in Explorer

### No Diagnostics Appearing

1. **Verify file type:**
   - Only `.go` files are analyzed
   - Check file is not in `ignored-files` patterns

2. **Check configuration:**
   - Ensure LSP server is enabled in settings
   - Try with verbose logging enabled

3. **Test CLI first:**
   ```bash
   unqueryvet ./your-file.go
   ```

### High Memory/CPU Usage

1. **Exclude large directories:**
   Add to `.unqueryvet.yaml`:
   ```yaml
   ignored-files:
     - "vendor/**"
     - "node_modules/**"
     - "**/*_generated.go"
   ```

2. **Disable advanced checks:**
   Remove `-n1` and `-sqli` flags if not needed

### GoLand Plugin Issues

1. **Plugin not loading:**
   - Check IDE version compatibility (2023.3+)
   - Reinstall the plugin
   - Clear caches: File → Invalidate Caches

2. **Settings not saving:**
   - Check write permissions
   - Try Settings Sync disable/enable

### VS Code Extension Issues

1. **Extension not activating:**
   - Check Output panel for errors
   - Reload window: `Ctrl+Shift+P` → "Reload Window"

2. **Conflicts with Go extension:**
   - Both can run together
   - Check that gopls is not blocking unqueryvet

---

## See Also

- [CLI Features Guide](CLI_FEATURES.md)
- [Custom Rules DSL](DSL.md)
- [Main README](../README.md)
