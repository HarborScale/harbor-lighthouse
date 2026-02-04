param(
    [switch]$Uninstall,
    [string]$Version = "v0.1.7" # Default version
)

$Repo = "harborscale/harbor-lighthouse"
$InstallDir = "C:\Program Files\HarborLighthouse"
$BinaryName = "lighthouse.exe"
$ExePath = Join-Path $InstallDir $BinaryName

# --- 0. ADMIN CHECK ---
$isWindowsAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isWindowsAdmin) {
    Write-Host "❌ Error: This script requires Administrator privileges." -ForegroundColor Red
    Write-Host "👉 Right-click PowerShell and select 'Run as Administrator'"
    exit 1
}

# --- 1. UNINSTALL LOGIC ---
if ($Uninstall) {
    Write-Host "🧹 Uninstalling Harbor Lighthouse..." -ForegroundColor Yellow

    # 1. Stop & Remove Service (if binary exists)
    if (Test-Path $ExePath) {
        Write-Host "   Stopping Service..."
        Start-Process -FilePath $ExePath -ArgumentList "--uninstall" -Wait -NoNewWindow
    }

    # 2. Cleanup Files
    if (Test-Path $InstallDir) {
        Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "   Removed Installation Directory."
    }

    # 3. Cleanup Path (Optional, but clean)
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($UserPath -like "*$InstallDir*") {
        $NewPath = $UserPath -replace [regex]::Escape("$InstallDir;"), ""
        $NewPath = $NewPath -replace [regex]::Escape("$InstallDir"), ""
        [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        Write-Host "   Cleaned PATH variable."
    }

    Write-Host "✅ Uninstalled successfully." -ForegroundColor Green
    return
}

# --- 2. DOWNLOAD ---
Write-Host "🚢 Installing Harbor Lighthouse $Version..." -ForegroundColor Cyan

# Create Directory
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

# Download URL (Assumes asset name format: lighthouse-windows-amd64.exe)
# Adjust specific naming if your release assets are zipped
$Url = "https://github.com/$Repo/releases/download/$Version/lighthouse-windows-amd64.exe"

Write-Host "⬇️  Downloading from $Url..."
try {
    Invoke-WebRequest -Uri $Url -OutFile $ExePath
} catch {
    Write-Host "❌ Download Failed: $_" -ForegroundColor Red
    exit 1
}

# --- 3. PATH SETUP ---
# Add to USER Path for future sessions
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "⚡ Added to User PATH (Future sessions)" -ForegroundColor DarkGray
}

# Add to CURRENT session environment so 'lighthouse' works immediately
if ($env:Path -notlike "*$InstallDir*") {
    $env:Path = "$env:Path;$InstallDir"
}

# --- 4. SERVICE INSTALL ---
Write-Host "⚙️  Registering Service..."
# Run the binary with --install flag
$proc = Start-Process -FilePath $ExePath -ArgumentList "--install" -Wait -NoNewWindow -PassThru

if ($proc.ExitCode -eq 0) {
    Write-Host "✅ Installed & Started Successfully!" -ForegroundColor Green
    Write-Host "👉 Configure it now: lighthouse --add --name 'pc-1' --harbor-id '123'" -ForegroundColor Cyan
} else {
    Write-Host "⚠️  Service install might have failed. Run 'lighthouse --install' manually to check." -ForegroundColor Yellow
}
