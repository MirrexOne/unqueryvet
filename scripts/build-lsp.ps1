# Build script for unqueryvet-lsp (PowerShell version)
# Builds binaries for all supported platforms

param(
    [string]$Version = "dev",
    [string]$OutputDir = "dist"
)

Write-Host "Building unqueryvet-lsp version: $Version" -ForegroundColor Green
Write-Host "Output directory: $OutputDir" -ForegroundColor Green

# Clean and create output directory
if (Test-Path $OutputDir) {
    Remove-Item -Recurse -Force $OutputDir
}
New-Item -ItemType Directory -Path $OutputDir | Out-Null

# Platforms to build for
$platforms = @(
    @{OS="windows"; Arch="amd64"},
    @{OS="windows"; Arch="arm64"},
    @{OS="linux"; Arch="amd64"},
    @{OS="linux"; Arch="arm64"},
    @{OS="darwin"; Arch="amd64"},
    @{OS="darwin"; Arch="arm64"}
)

# Build for each platform
foreach ($platform in $platforms) {
    $goos = $platform.OS
    $goarch = $platform.Arch

    $outputName = "unqueryvet-lsp-$goos-$goarch"
    if ($goos -eq "windows") {
        $outputName += ".exe"
    }

    $outputPath = Join-Path $OutputDir $outputName

    Write-Host "Building for $goos/$goarch..." -ForegroundColor Cyan

    $env:GOOS = $goos
    $env:GOARCH = $goarch

    $ldflags = "-s -w -X main.version=$Version"

    & go build -ldflags $ldflags -o $outputPath ./cmd/unqueryvet-lsp

    if ($LASTEXITCODE -eq 0) {
        $size = (Get-Item $outputPath).Length
        $sizeMB = [math]::Round($size / 1MB, 2)
        Write-Host "  Success Built: $outputName ($sizeMB MB)" -ForegroundColor Green
    } else {
        Write-Host "  Failed to build for $goos/$goarch" -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "Build complete! Binaries in: $OutputDir" -ForegroundColor Green
Write-Host ""
Get-ChildItem $OutputDir | Select-Object Name, @{Label="Size (MB)";Expression={[math]::Round($_.Length / 1MB, 2)}} | Format-Table
