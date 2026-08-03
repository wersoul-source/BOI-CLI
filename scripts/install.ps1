# BOI CLI Installer for Windows
# Usage: irm https://raw.githubusercontent.com/wersoul-source/BOI-CLI/main/scripts/install.ps1 | iex

$BOI_HOME = "$env:USERPROFILE\.boi"
$BOI_BIN = "$BOI_HOME\bin"

# Fetch latest release from GitHub
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/wersoul-source/BOI-CLI/releases/latest"
$VERSION = $release.tag_name -replace '^v',''

Write-Host "BOI CLI Installer" -ForegroundColor Magenta
Write-Host "Latest version: v$VERSION" -ForegroundColor Cyan
Write-Host ""

# 1. Create directories
New-Item -ItemType Directory -Force -Path $BOI_BIN | Out-Null
Write-Host "Install location: $BOI_BIN"

# 2. Download latest binary
$assetName = "boi_${VERSION}_windows_amd64.tar.gz"
$asset = $release.assets | Where-Object { $_.name -eq $assetName }
if (-not $asset) {
  Write-Host "ERROR: No asset found for $assetName" -ForegroundColor Red
  Write-Host "Available: $(($release.assets.name) -join ', ')"
  exit 1
}
Write-Host "Downloading $assetName..."
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile "$env:TEMP\boi.tar.gz"

# 3. Extract
Write-Host "Extracting..."
tar -xzf "$env:TEMP\boi.tar.gz" -C "$BOI_BIN"
Get-ChildItem "$BOI_BIN\boi_*" -Recurse -Filter "boi.exe" | Move-Item -Destination "$BOI_BIN\boi.exe" -Force
Remove-Item "$env:TEMP\boi.tar.gz"

# 4. Add to PATH
$path = [Environment]::GetEnvironmentVariable("Path", "User")
if ($path -notlike "*$BOI_BIN*") {
    [Environment]::SetEnvironmentVariable("Path", "$path;$BOI_BIN", "User")
    $env:Path += ";$BOI_BIN"
    Write-Host "Added to PATH"
}

# 5. Initialize
Write-Host "Initializing BOI CLI..."
& "$BOI_BIN\boi.exe" init

Write-Host ""
Write-Host "BOI CLI installed! Run 'boi' to start." -ForegroundColor Green
Write-Host "   Next: boi setup    - configure AI providers"
Write-Host "         boi          - launch TUI"
