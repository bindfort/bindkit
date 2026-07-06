#Requires -Version 5
# Packages the BindKit source into a clean release archive in dist/.
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

$version = (Select-String -Path cmd/server/main.go -Pattern 'const version = "(.*)"').Matches.Groups[1].Value
$out = "dist/bindkit-$version.zip"
New-Item -ItemType Directory -Force dist | Out-Null
if (Test-Path $out) { Remove-Item $out -Force }

Write-Host "Running tests before packaging..."
go test ./...
if ($LASTEXITCODE -ne 0) { throw "tests failed; not packaging" }

$staging = Join-Path $env:TEMP "bindkit-pkg-$version"
if (Test-Path $staging) { Remove-Item -Recurse -Force $staging }

$files = git ls-files --cached --others --exclude-standard
foreach ($file in $files) {
  if (-not (Test-Path -LiteralPath $file -PathType Leaf)) { continue }
  $target = Join-Path $staging $file
  New-Item -ItemType Directory -Force (Split-Path $target) | Out-Null
  Copy-Item -LiteralPath $file -Destination $target
}

Compress-Archive -Path "$staging/*" -DestinationPath $out -Force
Remove-Item -Recurse -Force $staging
Write-Host "packaged -> $out"
