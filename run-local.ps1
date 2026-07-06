$env:BINDKIT_TRANSPORT = 'http'
$env:BINDKIT_HTTP_ADDR = '127.0.0.1:8080'

@(
  'BINDKIT_AUTH_ENABLED',
  'BINDKIT_AUTH_MODE',
  'BINDKIT_API_KEYS',
  'BINDKIT_BILLING_ENABLED',
  'BINDKIT_PLAN_QUOTAS',
  'STRIPE_SECRET_KEY',
  'STRIPE_WEBHOOK_SECRET'
) | ForEach-Object {
  Remove-Item "Env:\$_" -ErrorAction SilentlyContinue
}

Set-Location $PSScriptRoot
go run ./cmd/server
