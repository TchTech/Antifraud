FROM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS builder
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
