# Deploy

## Docker

```bash
docker build -t bindkit .
docker run --rm -p 8080:8080 \
  -e BINDKIT_TRANSPORT=http \
  bindkit
```

Enable API-key auth and quotas when you are ready to test production controls:

```bash
docker run --rm -p 8080:8080 \
  -e BINDKIT_TRANSPORT=http \
  -e BINDKIT_AUTH_ENABLED=true \
  -e BINDKIT_API_KEYS=dev-key:free \
  -e BINDKIT_BILLING_ENABLED=true \
  -e BINDKIT_PLAN_QUOTAS=free:100,pro:10000 \
  bindkit
```

## Compose

```bash
docker compose up --build
```

## Fly

```bash
fly launch --copy-config
fly deploy
```

For authenticated deployments, set the auth environment variables from
[pricing.md](pricing.md) and store secrets with `fly secrets set`.
