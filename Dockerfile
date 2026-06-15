# ---- build: static, stripped binary ----
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./... \
 && CGO_ENABLED=0 GOOS=linux go build \
      -trimpath -ldflags="-s -w" \
      -o /out/bindkit ./cmd/server

# ---- runtime: distroless static, nonroot ----
# gcr.io/distroless/static is a ~2 MB base with no shell or package manager;
# combined with the stripped static binary the final image is well under 20 MB.
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/bindkit /usr/local/bin/bindkit
USER nonroot:nonroot
EXPOSE 8080
ENV BINDKIT_TRANSPORT=http \
    BINDKIT_HTTP_ADDR=:8080
ENTRYPOINT ["bindkit"]
