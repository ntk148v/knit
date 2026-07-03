#!/usr/bin/env pwsh
param(
    [string]$InstallDir = "$env:USERPROFILE\.local\bin"
)

$Repo = "ntk148v/knit"

# --- step 1: ensure npx (Node.js) ---
if (-not (Get-Command npx -ErrorAction SilentlyContinue)) {
    Write-Error "npx not found — install Node.js from https://nodejs.org"
    exit 1
}

Write-Host "==> Ensuring npx skills is available..."
& npx skills --version 2>$null

# --- step 2: detect architecture ---
$Arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture) {
    "X64"   { "amd64" }
    "Arm64" { "arm64" }
    default { throw "Unsupported architecture: $_" }
}

# --- step 3: resolve latest release tag ---
Write-Host "==> Fetching latest release..."
$api = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Version = $api.tag_name

# --- step 4: download & install ---
$Archive = "knit_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/$Version/$Archive"

Write-Host "==> Downloading knit $Version for windows/$Arch..."
$tmp = Join-Path $env:TEMP "knit-install"
New-Item -ItemType Directory -Path $tmp -Force | Out-Null
$zip = Join-Path $tmp $Archive
Invoke-WebRequest -Uri $Url -OutFile $zip

Write-Host "==> Extracting..."
Expand-Archive -Path $zip -DestinationPath $tmp -Force

$binary = Get-ChildItem -Path $tmp -Recurse -Filter "knit.exe" | Select-Object -First 1
if (-not $binary) {
    # fallback: try plain filename inside archive
    $binary = Get-ChildItem -Path $tmp -Recurse -Filter "knit" | Select-Object -First 1
}

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item $binary.FullName -Destination (Join-Path $InstallDir "knit.exe") -Force

Write-Host "==> Done! Make sure $InstallDir is in your PATH."

Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
