# Build the binary and run CI
FROM docker.io/library/golang:1.21-alpine AS builder

# Get certs
RUN apk --update add ca-certificates

# Copy local source
WORKDIR /app
COPY . .

# Build binary
RUN GOOS=linux go build -a -o quibbble cmd/main.go

# Build image
FROM scratch

# Copy certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy config and binary
WORKDIR /root/
COPY ./configs/quibbble.yaml ./configs/
COPY --from=builder /app/quibbble .

# Entry and port
CMD ["./quibbble"]
EXPOSE 8080
