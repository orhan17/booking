# --- Build stage: compile a static binary ---
FROM golang:1.23-alpine AS build

WORKDIR /src

# Download dependencies first so this layer caches unless go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Build. CGO disabled + trimpath + stripped symbols → a small, static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Run stage: minimal image with just the binary ---
FROM alpine:3.20

# ca-certificates for outbound TLS (future real operators); wget for healthcheck.
RUN apk add --no-cache ca-certificates wget \
    && adduser -D -u 10001 app

USER app
COPY --from=build /out/api /usr/local/bin/api

ENV ADDR=:8080
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=2s --retries=3 \
    CMD wget -qO- "http://localhost:8080/healthz" || exit 1

ENTRYPOINT ["/usr/local/bin/api"]
