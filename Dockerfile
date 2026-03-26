# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o karadul ./cmd/karadul

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates iptables ip6tables

COPY --from=builder /build/karadul /usr/local/bin/karadul

# Create data directory
RUN mkdir -p /var/lib/karadul

# Expose default ports
EXPOSE 8080/tcp
EXPOSE 3478/udp
EXPOSE 51820/udp

VOLUME ["/var/lib/karadul"]

ENTRYPOINT ["karadul"]
CMD ["--help"]
