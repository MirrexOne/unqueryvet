import * as vscode from "vscode";
import * as path from "path";
import * as fs from "fs";
import * as https from "https";
import { pipeline } from "stream";
import { promisify } from "util";

const streamPipeline = promisify(pipeline);

interface PlatformInfo {
  platform: string;
  arch: string;
  extension: string;
}

/**
 * Detect current platform and architecture
 */
export function detectPlatform(): PlatformInfo {
  const platform = process.platform;
  const arch = process.arch;

  let platformName: string;
  let archName: string;
  let extension: string;

  // Map Node.js platform to our release naming
  switch (platform) {
    case "win32":
      platformName = "windows";
      extension = ".exe";
      break;
    case "darwin":
      platformName = "darwin";
      extension = "";
      break;
    case "linux":
      platformName = "linux";
      extension = "";
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }

  // Map Node.js arch to our release naming
  switch (arch) {
    case "x64":
      archName = "amd64";
      break;
    case "arm64":
      archName = "arm64";
      break;
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }

  return {
    platform: platformName,
    arch: archName,
    extension,
  };
}

/**
 * Get the path where LSP server should be stored
 */
export function getLspPath(context: vscode.ExtensionContext): string {
  const platformInfo = detectPlatform();
  const binDir = path.join(context.globalStorageUri.fsPath, "bin");
  const lspName = `unqueryvet-lsp${platformInfo.extension}`;
  return path.join(binDir, lspName);
}

/**
 * Check if LSP server exists at the given path
 */
export function checkLspExists(lspPath: string): boolean {
  try {
    return fs.existsSync(lspPath);
  } catch {
    return false;
  }
}

/**
 * Get download URL for LSP server binary
 */
function getDownloadUrl(version: string): string {
  const platformInfo = detectPlatform();
  const binaryName = `unqueryvet-lsp-${platformInfo.platform}-${platformInfo.arch}${platformInfo.extension}`;
  return `https://github.com/MirrexOne/unqueryvet/releases/download/${version}/${binaryName}`;
}

/**
 * Download file with progress reporting
 */
async function downloadFile(
  url: string,
  destPath: string,
  onProgress?: (downloaded: number, total: number) => void,
): Promise<void> {
  return new Promise((resolve, reject) => {
    https
      .get(url, (response) => {
        // Handle redirects
        if (
          response.statusCode === 301 ||
          response.statusCode === 302 ||
          response.statusCode === 307
        ) {
          const redirectUrl = response.headers.location;
          if (redirectUrl) {
            downloadFile(redirectUrl, destPath, onProgress)
              .then(resolve)
              .catch(reject);
            return;
          }
        }

        if (response.statusCode !== 200) {
          reject(new Error(`Failed to download: HTTP ${response.statusCode}`));
          return;
        }

        const totalSize = parseInt(
          response.headers["content-length"] || "0",
          10,
        );
        let downloadedSize = 0;

        // Report progress
        response.on("data", (chunk) => {
          downloadedSize += chunk.length;
          if (onProgress && totalSize > 0) {
            onProgress(downloadedSize, totalSize);
          }
        });

        // Create destination directory if it doesn't exist
        const destDir = path.dirname(destPath);
        if (!fs.existsSync(destDir)) {
          fs.mkdirSync(destDir, { recursive: true });
        }

        // Write to file
        const fileStream = fs.createWriteStream(destPath);
        response.pipe(fileStream);

        fileStream.on("finish", () => {
          fileStream.close();
          // Make executable on Unix-like systems
          if (process.platform !== "win32") {
            fs.chmodSync(destPath, 0o755);
          }
          resolve();
        });

        fileStream.on("error", (err) => {
          fs.unlinkSync(destPath);
          reject(err);
        });
      })
      .on("error", (err) => {
        reject(err);
      });
  });
}

/**
 * Download and install LSP server
 */
export async function downloadLspServer(
  context: vscode.ExtensionContext,
  version: string = "v1.5.3",
): Promise<string> {
  const lspPath = getLspPath(context);
  const downloadUrl = getDownloadUrl(version);

  return vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: "Unqueryvet: Downloading LSP server",
      cancellable: false,
    },
    async (progress) => {
      try {
        progress.report({ message: "Starting download...", increment: 0 });

        await downloadFile(downloadUrl, lspPath, (downloaded, total) => {
          const percent = Math.round((downloaded / total) * 100);
          progress.report({
            message: `${(downloaded / 1024 / 1024).toFixed(1)} MB / ${(total / 1024 / 1024).toFixed(1)} MB`,
            increment: percent,
          });
        });

        progress.report({ message: "Download complete!", increment: 100 });
        return lspPath;
      } catch (error) {
        throw new Error(`Failed to download LSP server: ${error}`);
      }
    },
  );
}

/**
 * Find LSP server in common locations or download if not found
 */
export async function findOrDownloadLsp(
  context: vscode.ExtensionContext,
  config: vscode.WorkspaceConfiguration,
  outputChannel: vscode.OutputChannel,
): Promise<string | undefined> {
  // 1. Check custom path from settings
  let serverPath = config.get<string>("lspPath");
  if (serverPath && fs.existsSync(serverPath)) {
    outputChannel.appendLine(`Found LSP server at custom path: ${serverPath}`);
    return serverPath;
  }

  // 2. Check in extension's storage (previously downloaded)
  const storedLspPath = getLspPath(context);
  if (checkLspExists(storedLspPath)) {
    outputChannel.appendLine(
      `Found LSP server in extension storage: ${storedLspPath}`,
    );
    return storedLspPath;
  }

  // 3. Try to find in PATH
  const pathLsp = await findLspInPath();
  if (pathLsp) {
    outputChannel.appendLine(`Found LSP server in PATH: ${pathLsp}`);
    return pathLsp;
  }

  // 4. Try GOPATH/bin
  const gopathLsp = findLspInGopath();
  if (gopathLsp) {
    outputChannel.appendLine(`Found LSP server in GOPATH: ${gopathLsp}`);
    return gopathLsp;
  }

  // 5. LSP not found - prompt user to download
  outputChannel.appendLine("LSP server not found, offering to download...");

  const choice = await vscode.window.showInformationMessage(
    "Unqueryvet: LSP server not found. Would you like to download it automatically?",
    "Download",
    "Cancel",
    "Install Manually",
  );

  if (choice === "Download") {
    try {
      const downloadedPath = await downloadLspServer(context);
      vscode.window.showInformationMessage(
        "Unqueryvet: LSP server downloaded successfully!",
      );
      return downloadedPath;
    } catch (error) {
      vscode.window.showErrorMessage(`Failed to download LSP server: ${error}`);
      return undefined;
    }
  } else if (choice === "Install Manually") {
    vscode.window.showInformationMessage(
      "Install unqueryvet-lsp manually: go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet-lsp@latest",
    );
    return undefined;
  }

  return undefined;
}

/**
 * Find LSP in system PATH
 */
async function findLspInPath(): Promise<string | undefined> {
  const { exec } = require("child_process");
  const { promisify } = require("util");
  const execAsync = promisify(exec);

  const names = ["unqueryvet-lsp", "unqueryvet-lsp.exe"];

  for (const name of names) {
    try {
      const cmd = process.platform === "win32" ? "where" : "which";
      const { stdout } = await execAsync(`${cmd} ${name}`);
      const serverPath = stdout.trim().split("\n")[0];
      if (serverPath && fs.existsSync(serverPath)) {
        return serverPath;
      }
    } catch {
      // Not found, continue
    }
  }

  return undefined;
}

/**
 * Find LSP in GOPATH/bin
 */
function findLspInGopath(): string | undefined {
  const gopath = process.env.GOPATH || path.join(process.env.HOME || "", "go");
  const gopathBin = path.join(gopath, "bin", "unqueryvet-lsp");

  if (fs.existsSync(gopathBin)) {
    return gopathBin;
  }
  if (fs.existsSync(gopathBin + ".exe")) {
    return gopathBin + ".exe";
  }

  return undefined;
}
