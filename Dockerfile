FROM node:22-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

FROM golang:1.22-alpine AS go-builder
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod .
RUN go mod tidy
COPY . .
COPY --from=web-builder /web/dist ./internal/server/web
RUN CGO_ENABLED=0 go build -tags prod -o /build/subme ./cmd/subme

FROM node:22-alpine
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

# Default collectors (can be overridden by volume mount)
COPY collectors/ /app/default-collectors/

# Entrypoint script
COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY --from=go-builder /build/subme /usr/local/bin/subme

EXPOSE 9090
VOLUME ["/app/collectors", "/app/cache", "/app/data"]
ENTRYPOINT ["/entrypoint.sh"]
CMD ["serve", "--port", "9090", "--db", "/app/data/subme.db", "--cache", "/app/cache", "--collectors", "/app/collectors"]
