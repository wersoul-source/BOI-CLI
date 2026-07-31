# BOI CLI -- Windows Installer Script
# Usage: irm https://boi.sh/install.ps1 | iex
# หรือ:  powershell -ExecutionPolicy Bypass -File install.ps1

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:USERPROFILE\.boi\bin",
    [switch]$SkipChecksum = $false,
    [switch]$SkipInit = $false
)

$ErrorActionPreference = "Stop"

# ===============================================================
# STEP 0: Print Banner
# ===============================================================

Write-Host ""
Write-Host "  BOI CLI Installer" -ForegroundColor Cyan
Write-Host "  AI Agent Runtime -- BOI Family" -ForegroundColor DarkCyan
Write-Host "  Platform: Windows" -ForegroundColor DarkCyan
Write-Host ""

# ===============================================================
# STEP 1: Detect OS + Architecture
# ===============================================================

Write-Host "[1/8] Detecting environment..." -ForegroundColor Yellow

$OS = "Windows"
$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "x86" }

# PowerShell on ARM64?
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $Arch = "arm64"
}

Write-Host "  OS:   $OS" -ForegroundColor Gray
Write-Host "  Arch: $Arch" -ForegroundColor Gray

if ($Arch -eq "x86") {
    Write-Host "  Warning: 32-bit architecture. amd64 binary may not work." -ForegroundColor Yellow
}

# ===============================================================
# STEP 2: Resolve Version
# ===============================================================

Write-Host "[2/8] Resolving version..." -ForegroundColor Yellow

$ReleaseUrl = if ($Version -eq "latest") {
    "https://api.github.com/repos/wersoul-source/BOI-CLI/releases/latest"
} else {
    "https://api.github.com/repos/wersoul-source/BOI-CLI/releases/tags/$Version"
}

try {
    $ReleaseJson = Invoke-RestMethod -Uri $ReleaseUrl -Method Get -TimeoutSec 10
    $VersionTag = $ReleaseJson.tag_name
    Write-Host "  Version: $VersionTag" -ForegroundColor Gray
} catch {
    Write-Host "  Error: Could not fetch release info. Check your internet connection." -ForegroundColor Red
    Write-Host "  URL: $ReleaseUrl" -ForegroundColor Gray
    exit 1
}

# ===============================================================
# STEP 3: Build Download URL
# ===============================================================

Write-Host "[3/8] Building download URL..." -ForegroundColor Yellow

$BinaryName = "boi_Windows_${Arch}.zip"
$Asset = $ReleaseJson.assets | Where-Object { $_.name -eq $BinaryName }

if (-not $Asset) {
    Write-Host "  Error: No asset found matching: $BinaryName" -ForegroundColor Red
    Write-Host "  Available assets:" -ForegroundColor Gray
    foreach ($a in $ReleaseJson.assets) {
        Write-Host "    $($a.name)" -ForegroundColor Gray
    }
    exit 1
}

$DownloadUrl = $Asset.browser_download_url
Write-Host "  URL: $DownloadUrl" -ForegroundColor Gray

# ===============================================================
# STEP 4: Create Install Directory
# ===============================================================

Write-Host "[4/8] Creating install directory..." -ForegroundColor Yellow

if (-not (Test-Path -LiteralPath $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Host "  Created: $InstallDir" -ForegroundColor Gray
} else {
    Write-Host "  Directory exists: $InstallDir" -ForegroundColor Gray
}

# Backup existing binary if present
$BinaryPath = Join-Path $InstallDir "boi.exe"
$BackupPath = Join-Path $InstallDir "boi.old.exe"

if (Test-Path -LiteralPath $BinaryPath) {
    Write-Host "  Backing up existing binary..." -ForegroundColor Gray
    Copy-Item -LiteralPath $BinaryPath -Destination $BackupPath -Force
}

# ===============================================================
# STEP 5: Download Binary
# ===============================================================

Write-Host "[5/8] Downloading BOI CLI binary..." -ForegroundColor Yellow

$TempZip = Join-Path $env:TEMP "boi-cli-install.zip"
$TempExtract = Join-Path $env:TEMP "boi-cli-extract"

try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -TimeoutSec 120
    Write-Host "  Downloaded: $([math]::Round((Get-Item $TempZip).Length / 1MB, 1)) MB" -ForegroundColor Gray
} catch {
    Write-Host "  Error: Download failed. Check your internet connection." -ForegroundColor Red
    exit 1
}

# Extract
Write-Host "  Extracting..." -ForegroundColor Gray
if (Test-Path -LiteralPath $TempExtract) {
    Remove-Item -Recurse -Force -LiteralPath $TempExtract
}
Expand-Archive -LiteralPath $TempZip -DestinationPath $TempExtract -Force

# Move binary to install dir
$ExtractedBinary = Get-ChildItem -LiteralPath $TempExtract -Filter "boi.exe" -Recurse | Select-Object -First 1
if (-not $ExtractedBinary) {
    # Try without .exe extension
    $ExtractedBinary = Get-ChildItem -LiteralPath $TempExtract -Filter "boi" -Recurse | Select-Object -First 1
}

if (-not $ExtractedBinary) {
    Write-Host "  Error: Could not find boi.exe in extracted archive" -ForegroundColor Red
    exit 1
}

Copy-Item -LiteralPath $ExtractedBinary.FullName -Destination $BinaryPath -Force
Write-Host "  Installed: $BinaryPath" -ForegroundColor Gray

# Cleanup temp files
Remove-Item -Recurse -Force -LiteralPath $TempExtract -ErrorAction SilentlyContinue
Remove-Item -Force -LiteralPath $TempZip -ErrorAction SilentlyContinue

# ===============================================================
# STEP 6: Verify Checksum
# ===============================================================

Write-Host "[6/8] Verifying checksum..." -ForegroundColor Yellow

if (-not $SkipChecksum) {
    # Download SHA256SUMS.txt
    $ChecksumAsset = $ReleaseJson.assets | Where-Object { $_.name -eq "SHA256SUMS.txt" }
    
    if ($ChecksumAsset) {
        try {
            $ChecksumUrl = $ChecksumAsset.browser_download_url
            $ChecksumsText = Invoke-RestMethod -Uri $ChecksumUrl -Method Get -TimeoutSec 10
            
            # Find the line for our binary
            $ExpectedHash = $null
            foreach ($line in $ChecksumsText -split "`n") {
                if ($line -match "boi_Windows_${Arch}") {
                    $ExpectedHash = ($line -split '\s+')[0].Trim().ToLower()
                    break
                }
            }
            
            if ($ExpectedHash) {
                $ActualHash = (Get-FileHash -LiteralPath $BinaryPath -Algorithm SHA256).Hash.ToLower()
                
                if ($ActualHash -eq $ExpectedHash) {
                    Write-Host "  Checksum verified OK" -ForegroundColor Green
                } else {
                    Write-Host "  WARNING: Checksum mismatch!" -ForegroundColor Red
                    Write-Host "    Expected: $ExpectedHash" -ForegroundColor Gray
                    Write-Host "    Actual:   $ActualHash" -ForegroundColor Gray
                    Write-Host "  The binary may be corrupted or tampered with." -ForegroundColor Yellow
                    
                    if ($BackupPath -and (Test-Path -LiteralPath $BackupPath)) {
                        Write-Host "  Restoring previous version..." -ForegroundColor Yellow
                        Copy-Item -LiteralPath $BackupPath -Destination $BinaryPath -Force
                    }
                    exit 1
                }
            } else {
                Write-Host "  Warning: Could not find checksum entry for this platform" -ForegroundColor Yellow
            }
        } catch {
            Write-Host "  Warning: Could not verify checksum (network issue)" -ForegroundColor Yellow
        }
    } else {
        Write-Host "  Warning: No SHA256SUMS.txt found in release" -ForegroundColor Yellow
    }
} else {
    Write-Host "  Skipped (--SkipChecksum)" -ForegroundColor Gray
}

# ===============================================================
# STEP 7: Add to PATH
# ===============================================================

Write-Host "[7/8] Adding to PATH..." -ForegroundColor Yellow

$CurrentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
$CurrentMachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")

if ($CurrentUserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable(
        "Path",
        "$InstallDir;$CurrentUserPath",
        "User"
    )
    Write-Host "  Added to user PATH" -ForegroundColor Green
    
    # Also update current session
    $env:Path = "$InstallDir;$env:Path"
} else {
    Write-Host "  Already in PATH" -ForegroundColor Gray
}

# ===============================================================
# STEP 8: Initialize
# ===============================================================

Write-Host "[8/8] Initializing workspace..." -ForegroundColor Yellow

if (-not $SkipInit) {
    try {
        & $BinaryPath init --silent 2>$null
        Write-Host "  Workspace initialized" -ForegroundColor Green
    } catch {
        Write-Host "  Warning: Could not auto-initialize workspace" -ForegroundColor Yellow
        Write-Host "  Run 'boi init' manually after install" -ForegroundColor Gray
    }
} else {
    Write-Host "  Skipped (--SkipInit)" -ForegroundColor Gray
}

# ===============================================================
# SUCCESS
# ===============================================================

Write-Host ""
Write-Host "  ============================================" -ForegroundColor Green
Write-Host "   BOI CLI installed successfully!" -ForegroundColor Green
Write-Host "  ============================================" -ForegroundColor Green
Write-Host ""
Write-Host "  Version:  $VersionTag" -ForegroundColor Cyan
Write-Host "  Binary:   $BinaryPath" -ForegroundColor Cyan
Write-Host "  Config:   $env:USERPROFILE\.boi" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Quick start:" -ForegroundColor White
Write-Host "    boi              Launch TUI" -ForegroundColor Gray
Write-Host "    boi ask 'hello'  Test AI" -ForegroundColor Gray
Write-Host "    boi --help        All commands" -ForegroundColor Gray
Write-Host ""
Write-Host "  Next -- Set up LLM providers:" -ForegroundColor White
Write-Host "    cd to your project" -ForegroundColor Gray
Write-Host "    cp .env.example .env" -ForegroundColor Gray
Write-Host "    notepad .env  # Add PSC_1_API_KEY=..." -ForegroundColor Gray
Write-Host ""
Write-Host "  Docs: https://github.com/wersoul-source/BOI-CLI" -ForegroundColor DarkCyan
Write-Host ""
