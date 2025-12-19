$Version = "v0.0.7"
$Repo = "harborscale/harbor-lighthouse"
$Url = "https://github.com/$Repo/releases/download/$Version/lighthouse-windows.exe"
$InstallDir = "C:\Program Files\HarborLighthouse"
$ExePath = "$InstallDir\lighthouse.exe"

Write-Host "🚢 Installing Harbor Lighthouse $Version..." -ForegroundColor Cyan

# 1. Create Directory
if (!(Test-Path $InstallDir)) { New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null }

# 2. Download
Write-Host "⬇️  Downloading..."
Invoke-WebRequest -Uri $Url -OutFile $ExePath

# 3. Add to PATH (User Path)
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "✅ Added to PATH. Please restart your terminal." -ForegroundColor Yellow
} else {
    Write-Host "✅ Installed!" -ForegroundColor Green
}

Write-Host "👉 Run: lighthouse.exe --add --name 'pc' --harbor-id '123'"
