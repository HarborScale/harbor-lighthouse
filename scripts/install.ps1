param([switch]$Uninstall)

$Version = "v0.0.7"
$Repo = "harborscale/harbor-lighthouse"
$InstallDir = "C:\Program Files\HarborLighthouse"
$ExePath = "$InstallDir\lighthouse.exe"

# --- 🗑️ UNINSTALL MODE ---
if ($Uninstall) {
    Write-Host "🧹 Uninstalling Harbor Lighthouse..." -ForegroundColor Cyan

    # 1. Ask binary to remove Service (Stop & Delete)
    if (Test-Path $ExePath) {
        Start-Process -FilePath $ExePath -ArgumentList "--uninstall" -Wait -NoNewWindow
    }

    # 2. Remove Files
    if (Test-Path $InstallDir) {
        Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "✅ Files removed." -ForegroundColor Green
    }

    # 3. Clean PATH (Remove entry if it exists)
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($UserPath -like "*$InstallDir*") {
        $NewPath = ($UserPath -split ';' | Where-Object { $_ -ne $InstallDir }) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        Write-Host "✅ Removed from PATH." -ForegroundColor Green
    }

    Write-Host "✅ Uninstallation complete." -ForegroundColor Green
    return
}
# -------------------------

# --- 🚢 INSTALL MODE ---
Write-Host "🚢 Installing Harbor Lighthouse $Version..." -ForegroundColor Cyan
$Url = "https://github.com/$Repo/releases/download/$Version/lighthouse-windows.exe"

# 1. Create Directory
if (!(Test-Path $InstallDir)) { New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null }

# 2. Download
Write-Host "⬇️  Downloading..."
Invoke-WebRequest -Uri $Url -OutFile $ExePath

# 3. Add to PATH (User Path)
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "✅ Added to PATH. Please restart terminal." -ForegroundColor Yellow
} else {
    Write-Host "✅ Installed!" -ForegroundColor Green
}

Write-Host "👉 Run: lighthouse.exe --add --name 'pc' --harbor-id '123'"
