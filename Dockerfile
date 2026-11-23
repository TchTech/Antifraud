FROM golang:1.26.11-bookworm@sha256:dc40a13b61304647ed7e5d581d246b86413dfc29d63c6d018310feb93cd98fc0 AS builder
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
