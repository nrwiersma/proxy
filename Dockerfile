# Build container
FROM golang:1.12 as builder

ENV GO111MODULE=on

WORKDIR /app/
COPY ./ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -X main.version=$(git describe --tags --always)" -o proxy ./cmd/proxy

# Run container
FROM scratch

COPY --from=builder /app/proxy .
COPY --from=builder /app/config.yml .
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

EXPOSE 80
CMD ["./proxy", "server"]
