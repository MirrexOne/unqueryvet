import * as vscode from "vscode";
import * as path from "path";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";
import { StatusBarManager } from "./statusBar";

let client: LanguageClient | undefined;
let outputChannel: vscode.OutputChannel;
let statusBar: StatusBarManager;

export async function activate(context: vscode.ExtensionContext) {
  outputChannel = vscode.window.createOutputChannel("Unqueryvet");
  outputChannel.appendLine("Unqueryvet extension activating...");

  // Create status bar
  statusBar = new StatusBarManager();
  context.subscriptions.push({ dispose: () => statusBar.dispose() });

  const config = vscode.workspace.getConfiguration("unqueryvet");

  if (!config.get<boolean>("enable")) {
    outputChannel.appendLine("Unqueryvet is disabled in settings");
    statusBar.setDisabled();
    return;
  }

  // Start LSP client
  try {
    await startLanguageClient(context, config);
  } catch (error) {
    outputChannel.appendLine(`Failed to start LSP client: ${error}`);
    statusBar.setError("LSP not found");
    vscode.window.showErrorMessage(
      `Unqueryvet: Failed to start language server. ${error}`,
    );
  }

  // Register commands
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "unqueryvet.analyzeFile",
      analyzeCurrentFile,
    ),
    vscode.commands.registerCommand(
      "unqueryvet.analyzeWorkspace",
      analyzeWorkspace,
    ),
    vscode.commands.registerCommand("unqueryvet.fixAll", fixAllIssues),
    vscode.commands.registerCommand("unqueryvet.showOutput", () =>
      outputChannel.show(),
    ),
    vscode.commands.registerCommand(
      "unqueryvet.restart",
      restartLanguageServer,
    ),
  );

  // Listen for diagnostics changes to update status bar
  context.subscriptions.push(
    vscode.languages.onDidChangeDiagnostics(updateStatusBar),
  );

  // Listen for config changes
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("unqueryvet")) {
        handleConfigChange();
      }
    }),
  );

  outputChannel.appendLine("Unqueryvet extension activated");
}

async function startLanguageClient(
  context: vscode.ExtensionContext,
  config: vscode.WorkspaceConfiguration,
): Promise<void> {
  // Find the LSP server executable
  let serverPath = config.get<string>("lspPath");

  if (!serverPath) {
    // Try to find in PATH or common locations
    serverPath = await findLspServer();
  }

  if (!serverPath) {
    throw new Error(
      "unqueryvet-lsp not found. Please install it or set unqueryvet.lspPath",
    );
  }

  outputChannel.appendLine(`Using LSP server: ${serverPath}`);

  const serverOptions: ServerOptions = {
    run: {
      command: serverPath,
      transport: TransportKind.stdio,
    },
    debug: {
      command: serverPath,
      transport: TransportKind.stdio,
    },
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "go" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*.go"),
    },
    outputChannel: outputChannel,
    initializationOptions: {
      checkN1Queries: config.get<boolean>("checkN1Queries"),
      checkSQLInjection: config.get<boolean>("checkSQLInjection"),
      severity: config.get<string>("severity"),
    },
  };

  client = new LanguageClient(
    "unqueryvet",
    "Unqueryvet Language Server",
    serverOptions,
    clientOptions,
  );

  await client.start();
  outputChannel.appendLine("LSP client started successfully");
}

async function findLspServer(): Promise<string | undefined> {
  const { exec } = require("child_process");
  const { promisify } = require("util");
  const execAsync = promisify(exec);

  // Try common names
  const names = ["unqueryvet-lsp", "unqueryvet-lsp.exe"];

  for (const name of names) {
    try {
      // Check if in PATH
      const cmd = process.platform === "win32" ? "where" : "which";
      const { stdout } = await execAsync(`${cmd} ${name}`);
      const serverPath = stdout.trim().split("\n")[0];
      if (serverPath) {
        return serverPath;
      }
    } catch {
      // Not found, continue
    }
  }

  // Try GOPATH/bin
  const gopath = process.env.GOPATH || path.join(process.env.HOME || "", "go");
  const gopathBin = path.join(gopath, "bin", "unqueryvet-lsp");

  try {
    const fs = require("fs");
    if (fs.existsSync(gopathBin)) {
      return gopathBin;
    }
    if (fs.existsSync(gopathBin + ".exe")) {
      return gopathBin + ".exe";
    }
  } catch {
    // Not found
  }

  return undefined;
}

async function analyzeCurrentFile(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage("No active editor");
    return;
  }

  if (editor.document.languageId !== "go") {
    vscode.window.showWarningMessage("Current file is not a Go file");
    return;
  }

  outputChannel.appendLine(`Analyzing: ${editor.document.fileName}`);

  // Force document sync if client is available
  if (client) {
    await client.sendNotification("textDocument/didSave", {
      textDocument: { uri: editor.document.uri.toString() },
    });
    vscode.window.showInformationMessage("Analysis complete");
  } else {
    vscode.window.showErrorMessage("Language server not running");
  }
}

async function analyzeWorkspace(): Promise<void> {
  const workspaceFolders = vscode.workspace.workspaceFolders;
  if (!workspaceFolders) {
    vscode.window.showWarningMessage("No workspace folder open");
    return;
  }

  outputChannel.appendLine("Analyzing workspace...");

  // Find all Go files
  const goFiles = await vscode.workspace.findFiles("**/*.go", "**/vendor/**");
  outputChannel.appendLine(`Found ${goFiles.length} Go files`);

  vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: "Unqueryvet: Analyzing workspace",
      cancellable: true,
    },
    async (progress, token) => {
      let analyzed = 0;
      for (const file of goFiles) {
        if (token.isCancellationRequested) {
          break;
        }

        progress.report({
          message: `${analyzed}/${goFiles.length} files`,
          increment: 100 / goFiles.length,
        });

        // Open and analyze each file
        const doc = await vscode.workspace.openTextDocument(file);
        if (client) {
          await client.sendNotification("textDocument/didOpen", {
            textDocument: {
              uri: doc.uri.toString(),
              languageId: "go",
              version: doc.version,
              text: doc.getText(),
            },
          });
        }
        analyzed++;
      }

      return new Promise<void>((resolve) => {
        setTimeout(() => {
          vscode.window.showInformationMessage(`Analyzed ${analyzed} files`);
          resolve();
        }, 500);
      });
    },
  );
}

async function fixAllIssues(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage("No active editor");
    return;
  }

  if (!client) {
    vscode.window.showErrorMessage("Language server not running");
    return;
  }

  outputChannel.appendLine("Requesting code actions for fix all...");

  // Get all diagnostics for current file
  const diagnostics = vscode.languages.getDiagnostics(editor.document.uri);
  const unqueryvetDiagnostics = diagnostics.filter(
    (d) => d.source === "unqueryvet",
  );

  if (unqueryvetDiagnostics.length === 0) {
    vscode.window.showInformationMessage("No Unqueryvet issues to fix");
    return;
  }

  // Apply fixes for each diagnostic
  let fixed = 0;
  for (const diagnostic of unqueryvetDiagnostics) {
    try {
      const codeActions = await vscode.commands.executeCommand<
        vscode.CodeAction[]
      >(
        "vscode.executeCodeActionProvider",
        editor.document.uri,
        diagnostic.range,
      );

      if (codeActions && codeActions.length > 0) {
        const quickFix = codeActions.find((a) =>
          a.kind?.contains(vscode.CodeActionKind.QuickFix),
        );
        if (quickFix && quickFix.edit) {
          await vscode.workspace.applyEdit(quickFix.edit);
          fixed++;
        }
      }
    } catch (error) {
      outputChannel.appendLine(`Failed to fix: ${error}`);
    }
  }

  vscode.window.showInformationMessage(
    `Fixed ${fixed}/${unqueryvetDiagnostics.length} issues`,
  );
}

function updateStatusBar(): void {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "go") {
    return;
  }

  const diagnostics = vscode.languages.getDiagnostics(editor.document.uri);
  const unqueryvetDiagnostics = diagnostics.filter(
    (d) => d.source === "unqueryvet",
  );
  statusBar.setIssueCount(unqueryvetDiagnostics.length);
}

async function handleConfigChange(): Promise<void> {
  const config = vscode.workspace.getConfiguration("unqueryvet");

  if (!config.get<boolean>("enable")) {
    statusBar.setDisabled();
    if (client) {
      await client.stop();
      client = undefined;
    }
  } else {
    // Restart LSP with new config
    await restartLanguageServer();
  }
}

async function restartLanguageServer(): Promise<void> {
  outputChannel.appendLine("Restarting language server...");

  if (client) {
    await client.stop();
    client = undefined;
  }

  const config = vscode.workspace.getConfiguration("unqueryvet");
  try {
    // Get extension context - we need to pass it properly
    const ext = vscode.extensions.getExtension("unqueryvet.unqueryvet");
    if (ext) {
      await startLanguageClient(ext.extensionUri as any, config);
      vscode.window.showInformationMessage(
        "Unqueryvet: Language server restarted",
      );
    }
  } catch (error) {
    statusBar.setError("Restart failed");
    vscode.window.showErrorMessage(`Failed to restart: ${error}`);
  }
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
