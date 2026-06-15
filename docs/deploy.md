# Deploy

## Docker

```bash
docker build -t bindkit .
docker run --rm -p 8080:8080 \
  -e BINDKIT_TRANSPORT=http \
  -e BINDKIT_AUTH_ENABLED=true \
  -e BINDKIT_API_KEYS=dev-key:free \
  bindkit
```

## Compose

```bash
docker compose up --build
```

## Fly

```bash
fly launch --copy-config
fly secrets set BINDKIT_API_KEYS=dev-key:free
fly deploy
```

