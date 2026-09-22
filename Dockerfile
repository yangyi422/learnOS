FROM node:22.18-bookworm-slim AS web-builder
WORKDIR /src/web
RUN npm install -g npm@11.18.0
COPY web/package.json ./
RUN npm install --include=optional
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

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl tzdata \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-builder /out/learnos /app/learnos
RUN mkdir -p /app/data
RUN mkdir -p /app/backups
ENV APP_ADDR=:8080 \
    APP_DATA_DIR=/app/data \
    APP_ENV=production
EXPOSE 8080
VOLUME ["/app/data", "/app/backups"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl --fail --silent http://127.0.0.1:8080/health >/dev/null || exit 1
ENTRYPOINT ["/app/learnos"]
