# BUILD stage

FROM golang:1.25.2-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o controller ./cmd/main.go

# Runtime

FROM ubuntu:24.04

WORKDIR /

COPY --from=builder /build/controller /controller

RUN apt-get update && apt-get install -y tcpdump && rm -rf /var/lib/apt/lists/*

CMD ["./controller"]