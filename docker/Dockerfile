# ==========================================
# Stage 1: Build React Frontend
# ==========================================
FROM node:22-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ==========================================
# Stage 2: Build Go Application with Embedded UI
# ==========================================
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy production frontend build assets into embedded path
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o stream ./cmd

# ==========================================
# Stage 3: Production Runtime with FFmpeg
# ==========================================
FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ffmpeg \
    ca-certificates \
    tzdata && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy statically compiled stream binary
COPY --from=go-builder /app/stream /usr/local/bin/stream

# Setup unified storage root
ENV DATA_DIR=/app/data
RUN mkdir -p /app/data
VOLUME ["/app/data"]

EXPOSE 8068

ENTRYPOINT ["/usr/local/bin/stream"]
CMD ["--data-dir", "/app/data"]
