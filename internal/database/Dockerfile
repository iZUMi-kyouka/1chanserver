# Stage 1: Build
FROM golang:1.20-alpine AS builder

# Set up environment variables
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go app
RUN chmod +x run_server.sh
RUN go build -o server ./cmd/server/main.go

# Stage 2: Runtime
FROM alpine:latest

# Install any necessary runtime dependencies
RUN apk --no-cache add ca-certificates

# Set the working directory inside the container
WORKDIR /root/

# Copy the built binary and required files from the build stage
COPY --from=builder /app/server .
COPY --from=builder /app/public ./public
COPY --from=builder /app/frontend ./frontend
COPY --from=builder /app/*.sql ./
COPY --from=builder /app/*.sh ./

# Expose the port your app listens on
EXPOSE 8080

# Command to run the app
CMD ["./run_server.sh"]