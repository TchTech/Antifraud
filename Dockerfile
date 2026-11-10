FROM golang:1.26.10-bookworm@sha256:5de7c99eb78ca5e515a4c340235d0955ec5964a5071c492c15503d7c267da473 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/gateway ./cmd/gateway && CGO_ENABLED=0 go build -o /out/risk ./cmd/risk && CGO_ENABLED=0 go build -o /out/network-collector ./cmd/network-collector
FROM debian:bookworm-slim@sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc
WORKDIR /app
RUN useradd --system demo
COPY --from=builder /out/ /app/
COPY db/migrations/ /app/db/migrations/
USER demo
CMD ["/app/gateway"]
