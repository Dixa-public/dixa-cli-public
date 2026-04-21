$ErrorActionPreference = "Stop"

$Repo = if ($env:DIXA_REPO) { $env:DIXA_REPO } else { "Dixa-public/dixa-cli-public" }

function Get-LatestTag {
  $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
  return $release.tag_name
}

$Tag = if ($env:DIXA_VERSION) { $env:DIXA_VERSION } else { Get-LatestTag }
if (-not $Tag) {
  throw "Failed to resolve the latest release tag."
}

if ($Tag.StartsWith("v")) {
  $Version = $Tag.Substring(1)
} else {
  $Version = $Tag
  $Tag = "v$Tag"
}

$Arch = switch ($env:PROCESSOR_ARCHITECTURE.ToLower()) {
  "arm64" { "arm64" }
  default { "amd64" }
}

$InstallDir = if ($env:INSTALL_DIR) {
  $env:INSTALL_DIR
} else {
  Join-Path $env:LOCALAPPDATA "Programs\dixa\bin"
}

$TmpDir = Join-Path $env:TEMP "dixa-install-$Version"
New-Item -ItemType Directory -Force -Path $TmpDir | Out-Null
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$ArchiveName = "dixa_${Version}_windows_${Arch}.zip"
$ArchivePath = Join-Path $TmpDir $ArchiveName
$ArchiveUrl = "https://github.com/$Repo/releases/download/$Tag/$ArchiveName"

Invoke-WebRequest -Uri $ArchiveUrl -OutFile $ArchivePath
Expand-Archive -Force -Path $ArchivePath -DestinationPath $TmpDir
Copy-Item -Force (Join-Path $TmpDir "dixa.exe") (Join-Path $InstallDir "dixa.exe")

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
$PathEntries = @()
if ($UserPath) {
  $PathEntries = $UserPath.Split(";") | Where-Object { $_ }
}

if ($PathEntries -notcontains $InstallDir) {
  $NewPath = if ($UserPath) { "$UserPath;$InstallDir" } else { $InstallDir }
  [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
}

$env:DIXA_BIN = Join-Path $InstallDir "dixa.exe"
Write-Host ""
Write-Host "Installed dixa $Version to $env:DIXA_BIN"
& $env:DIXA_BIN --version
Write-Host ""
Write-Host "If this is a new install, open a new PowerShell window before running 'dixa'."
