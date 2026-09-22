# ---------- stage 1: build ----------
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Copy only the dependency files first: this layer is cached
# and only rebuilt when go.mod or go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 builds a static binary that runs on an image without libc.
# -trimpath strips your machine's paths from the binary.
# -s -w drop the symbol table and DWARF, making the binary smaller.
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/api \
        ./cmd/api

# ---------- stage 2: runtime ----------
FROM alpine:3.20

# ca-certificates: needed if the API calls HTTPS endpoints (e.g. Anvisa leaflets).
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/api /app/api

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/api"]
