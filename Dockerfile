FROM node:22.18-bookworm-slim AS web-builder
WORKDIR /src/web
RUN npm install -g npm@11.18.0
COPY web/package.json web/package-lock.json ./
RUN npm ci --include=optional
COPY web/ ./
RUN npm run build

FROM golang:1.26.5-bookworm AS go-builder
WORKDIR /src
COPY go.mod go.sum ./

ARG GOPROXY=https://goproxy.cn|direct
ENV GOPROXY=${GOPROXY}

RUN go mod download
COPY . .
COPY --from=web-builder /src/internal/webassets/dist ./internal/webassets/dist
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/learnos ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/healthcheck ./cmd/healthcheck

FROM debian:bookworm-slim
# The runtime image intentionally runs no apt: the CA bundle is copied from the
# Go builder and the health check is a compiled Go binary. This keeps production
# builds independent of Debian mirrors (a flaky mirror broke a CI deploy).
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=go-builder /out/learnos /app/learnos
COPY --from=go-builder /out/healthcheck /app/healthcheck
WORKDIR /app
RUN mkdir -p /app/data /app/backups
ENV APP_ADDR=:8080 \
    APP_DATA_DIR=/app/data \
    APP_ENV=production
EXPOSE 8080
VOLUME ["/app/data", "/app/backups"]
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD ["/app/healthcheck"]
ENTRYPOINT ["/app/learnos"]
