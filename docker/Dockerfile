# Use the official Golang image to create a build artifact.
FROM golang:1.22 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app as a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o stream cmd/*.go


# Actual exposed image
# Start a new stage from scratch
FROM ubuntu:latest

# Install FFmpeg with CUDA support
RUN apt-get update && \
    apt-get install -y ffmpeg && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/stream /usr/local/bin/stream

# Copy the config file
COPY config.json /app/config.json

# Expose port 8068 to the outside world
EXPOSE 8068

# Command to run the executable
CMD ["stream", "--config", "/app/config.json"]
