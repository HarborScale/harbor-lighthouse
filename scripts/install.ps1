param([switch]$Uninstall)
$Version = "v0.0.9"
$Repo = "harborscale/harbor-lighthouse"
$InstallDir = "C:\Program Files\HarborLighthouse"
$ExePath = "$InstallDir\lighthouse.exe"

if ($Uninstall) {
    if (Test-Path $ExePath) { Start-Process -FilePath $ExePath -ArgumentList "--uninstall" -Wait -NoNewWindow }
    if (Test-Path $InstallDir) { Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue }
    Write-Host "✅ Uninstalled." -ForegroundColor Green
    return
}

Write-Host "🚢 Installing Harbor Lighthouse $Version..." -ForegroundColor Cyan
$Url = "https://github.com/$Repo/releases/download/$Version/lighthouse-windows.exe"

if (!(Test-Path $InstallDir)) { New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null }
Invoke-WebRequest -Uri $Url -OutFile $ExePath

# Add to PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
}

# --- ⚡ INSTALL SERVICE NOW ---
Write-Host "⚙️  Registering Service..."
Start-Process -FilePath $ExePath -ArgumentList "--install" -Wait -NoNewWindow

Write-Host "✅ Installed & Running (Idle)" -ForegroundColor Green
Write-Host "👉 Now configure it: lighthouse.exe --add --name 'pc-1' --harbor-id '123'"
