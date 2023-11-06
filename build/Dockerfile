# Build the binary and run CI
FROM golang:1.21-alpine3.18 AS builder

# Copy local source
WORKDIR /app
COPY . .

# Build binary
RUN GOOS=linux go build -a -o quibbble cmd/main.go

# Build image
FROM scratch

# Copy config and binary
WORKDIR /root/
COPY ./configs/quibbble.yaml ./configs/
COPY --from=builder /app/quibbble .

# Entry and port
CMD ["./quibbble"]
EXPOSE 8080
