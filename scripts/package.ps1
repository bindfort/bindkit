#Requires -Version 5
# Packages the Bindkit source into a clean, customer-ready zip in dist/.
# Excludes git, build output, logs, and internal dev-process files.
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

# /XD excludes dirs, /XF excludes files. robocopy exit codes < 8 are success.
robocopy . $staging /E `
  /XD .git dist landingpage .github\..cache `
  /XF *.exe *.log agent.md handoff.json tasks.md architecture-map.html `
  | Out-Null
if ($LASTEXITCODE -ge 8) { throw "robocopy failed ($LASTEXITCODE)" }

Compress-Archive -Path "$staging/*" -DestinationPath $out -Force
Remove-Item -Recurse -Force $staging
Write-Host "packaged -> $out"
