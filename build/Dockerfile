# Build the binary and run CI
FROM golang:1.18-alpine AS builder

RUN apk update && apk upgrade

# Intall gcc
RUN apk add build-base

# Copy local source
WORKDIR /app
COPY . .

# Pull dependencies
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.31.0

# Build binary
RUN GOOS=linux go build -a -o quibbble cmd/main.go

# Build image
FROM alpine:3.14

RUN apk update && apk upgrade

# Copy config and binary
WORKDIR /root/
COPY ./configs/quibbble.yaml ./configs/
COPY --from=builder /app/quibbble .

# Entry and port
CMD ["./quibbble"]
EXPOSE 8080