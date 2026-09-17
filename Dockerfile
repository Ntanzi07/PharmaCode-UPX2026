# ---------- stage 1: build ----------
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Copia so os arquivos de dependencia primeiro: essa camada fica em cache
# e so e refeita quando go.mod ou go.sum mudam.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 gera um binario estatico, que roda numa imagem sem libc.
# -trimpath remove caminhos da sua maquina do binario.
# -s -w tiram a tabela de simbolos e o DWARF, deixando o binario menor.
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/api \
        ./cmd/api

# ---------- stage 2: runtime ----------
FROM alpine:3.20

# ca-certificates: necessario se a API for chamar HTTPS (ex: bula da Anvisa).
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/api /app/api

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/api"]
