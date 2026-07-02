# install.ps1 — SupCode 一键安装脚本 (Windows)
param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:LOCALAPPDATA\supcode\bin"
)

$Repo = "supcode/supcode"

Write-Host "==> SupCode Installer for Windows" -ForegroundColor Cyan

# ── Resolve version ──────────────────────────────────────
if ($Version -eq "latest") {
    $apiUrl = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $release = Invoke-RestMethod -Uri $apiUrl -ErrorAction Stop
        $Version = $release.tag_name.TrimStart('v')
    } catch {
        Write-Error "Failed to fetch latest version: $_"
        exit 1
    }
}

# ── Create install dir ───────────────────────────────────
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# ── Detect arch ─────────────────────────────────────────
$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "x86_64" }
    "ARM64" { "arm64" }
    default {
        Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
        exit 1
    }
}

# ── Download ────────────────────────────────────────────
$ArchiveName = "supcode_${Version}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/v${Version}/${ArchiveName}"
$ChecksumUrl = "${DownloadUrl}.sha256"

$TmpDir = Join-Path $env:TEMP "supcode_install_$(Get-Random)"
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

$ZipPath = Join-Path $TmpDir "archive.zip"

Write-Host "==> Downloading $DownloadUrl"
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -ErrorAction Stop
} catch {
    Write-Error "Download failed: $_"
    exit 1
}

# ── Verify checksum ─────────────────────────────────────
try {
    $expectedHash = (Invoke-WebRequest -Uri $ChecksumUrl -ErrorAction Stop).Content.Trim().Split(' ')[0]
    $actualHash = (Get-FileHash -Path $ZipPath -Algorithm SHA256).Hash.ToLower()
    if ($expectedHash -ne $actualHash) {
        Write-Error "Checksum mismatch! Expected $expectedHash, got $actualHash"
        exit 1
    }
    Write-Host "==> Checksum verified" -ForegroundColor Green
} catch {
    Write-Warning "Checksum verification skipped: $_"
}

# ── Extract ─────────────────────────────────────────────
Write-Host "==> Extracting"
Expand-Archive -Path $ZipPath -DestinationPath $TmpDir -Force

# ── Install ─────────────────────────────────────────────
$ExeSrc = Join-Path $TmpDir "supcode.exe"
$ExeDst = Join-Path $InstallDir "supcode.exe"

Move-Item -Path $ExeSrc -Destination $ExeDst -Force

# ── Add to PATH ─────────────────────────────────────────
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$InstallDir*") {
    $newPath = "$InstallDir;$userPath"
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    $env:PATH = "$InstallDir;$env:PATH"
    Write-Host "==> Added $InstallDir to PATH" -ForegroundColor Green
}

# ── Cleanup ─────────────────────────────────────────────
Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue

Write-Host "==> Installed supcode v${Version} to ${ExeDst}" -ForegroundColor Green
Write-Host "    Run 'supcode --help' to get started." -ForegroundColor Cyan

# ── Verify ──────────────────────────────────────────────
$installed = Get-Command supcode -ErrorAction SilentlyContinue
if (-not $installed) {
    Write-Warning "supcode not found in PATH. You may need to restart your terminal."
}
else {
    Write-Host "    $(supcode --version 2>$null)" -ForegroundColor Gray
}
