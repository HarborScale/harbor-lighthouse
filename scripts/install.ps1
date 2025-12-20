param([switch]$Uninstall)
$Version = "v0.1.0"
$Repo = "harborscale/harbor-lighthouse"
$InstallDir = "C:\Program Files\HarborLighthouse"
$ExePath = "$InstallDir\lighthouse.exe"

# --- 1. UNINSTALL LOGIC ---
if ($Uninstall) {
    if (Test-Path $ExePath) { 
        # Attempt to stop/uninstall service first
        Start-Process -FilePath $ExePath -ArgumentList "--uninstall" -Wait -NoNewWindow 
    }
    if (Test-Path $InstallDir) { 
        Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue 
    }
    Write-Host "✅ Uninstalled." -ForegroundColor Green
    return
}

# --- 2. DOWNLOAD ---
Write-Host "🚢 Installing Harbor Lighthouse $Version..." -ForegroundColor Cyan
# NOTE: Ensure this URL is reachable or file exists for testing
$Url = "https://github.com/$Repo/releases/download/$Version/lighthouse-windows.exe"

if (!(Test-Path $InstallDir)) { New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null }
# For testing locally, you might want to Copy-Item instead of Download if building locally
# Copy-Item ".\lighthouse.exe" -Destination $ExePath
Invoke-WebRequest -Uri $Url -OutFile $ExePath

# --- 3. PATH SETUP  ---
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    # Update Registry for FUTURE sessions
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
}

# Update CURRENT session so it works immediately
if ($env:Path -notlike "*$InstallDir*") {
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "⚡ Added to current PATH." -ForegroundColor DarkGray
}

# --- 4. SERVICE INSTALL ---
Write-Host "⚙️  Registering Service..."
# We use $ExePath specifically to ensure we call the installed version
Start-Process -FilePath $ExePath -ArgumentList "--install" -Wait -NoNewWindow

Write-Host "✅ Installed & Running (Idle)" -ForegroundColor Green
Write-Host "👉 Now configure it: lighthouse --add --name 'pc-1' --harbor-id '123'"
