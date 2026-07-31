# -----------------------------------------------------------------------------
# STAGE 1: Builder
# -----------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

# Install zig for cross-compiling the C runtime with musl
# Alpine's package repository provides zig directly.
RUN apk add --no-cache zig make git

WORKDIR /app

# Copy project source
COPY . .

# Build the Wolf compiler CLI locally
RUN go build -o /usr/local/bin/wolf ./cmd/wolf

# Build the native bench_server binary statically
# Using zig cc under the hood (via wolf build --static)
RUN wolf build --static examples/bench_server.wolf

# -----------------------------------------------------------------------------
# STAGE 2: Runner (Zero-Dependency)
# -----------------------------------------------------------------------------
FROM scratch

# Copy the statically compiled native binary from the builder stage
COPY --from=builder /app/wolf_out/bench_server /server

# Expose standard Wolf API port
EXPOSE 8080

# Run the compiled static server
CMD ["/server"]
