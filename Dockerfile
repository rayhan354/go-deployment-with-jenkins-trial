# Build stage
FROM golang:1.21-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

# Copy main.go
COPY main.go .

# Initialize a Go module (this creates go.mod)
RUN go mod init app

# Build with version injection
ARG VERSION=dev
RUN go build -ldflags="-X main.version=${VERSION}" -o app .

# Final stage
FROM scratch
COPY --from=builder /app/app /app
EXPOSE 8080
CMD ["/app"]
