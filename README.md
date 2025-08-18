# Microservice Template

Check the example/main.go  and the example/ folder in how to use in your own microservices

## Proposed Dockerfile
~~~Dockerfile
# Stage 1: Build stage
FROM docker.io/library/golang:1.24.4 AS builder

# Set environment variables for the build
WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go binary
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o service cmd/main.go

# Stage 2: Run stage (minimal image)
FROM gcr.io/distroless/static:nonroot

# Set a non-root user
USER nonroot:nonroot

# Set working directory and copy built binary
WORKDIR /app

# Copy the .env.template and releasenotes.txt files - IMPORTANT: bootstrap.json is injected via configmap
COPY --from=builder /app/.env.toml .env
COPY --from=builder /app/releasenotes.txt releasenotes.txt


# Copy the binary from the builder stage
COPY --from=builder /app/service .


# Expose port 8001 for main service and 9091 for metrics
EXPOSE 8001 9091

# Command to run the binary
CMD ["./service"]
~~~