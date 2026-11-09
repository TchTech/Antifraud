FROM golang:1.25.7-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/gateway ./cmd/gateway && CGO_ENABLED=0 go build -o /out/risk ./cmd/risk && CGO_ENABLED=0 go build -o /out/network-collector ./cmd/network-collector
FROM debian:bookworm-slim
WORKDIR /app
RUN useradd --system demo
COPY --from=builder /out/ /app/
COPY db/migrations/ /app/db/migrations/
USER demo
CMD ["/app/gateway"]
