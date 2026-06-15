$env:BINDKIT_TRANSPORT = 'http'
$env:BINDKIT_HTTP_ADDR = '127.0.0.1:8080'
$env:BINDKIT_AUTH_ENABLED = 'true'
$env:BINDKIT_API_KEYS = 'dev-key:free'
$env:BINDKIT_BILLING_ENABLED = 'true'
$env:BINDKIT_PLAN_QUOTAS = 'free:100'

Set-Location $PSScriptRoot
.\bindkit-local.exe
