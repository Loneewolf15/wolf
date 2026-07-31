# -----------------------------------------------------------------------------
# STAGE 1: Builder
# -----------------------------------------------------------------------------
FROM golang:1.22-bookworm AS builder

# Install build dependencies (Wolf requires clang/llvm, liburing, mariadb)
# We use clang-15/llvm-15 because LLVM 15+ supports opaque pointers by default,
# which the Wolf compiler emits. Debian Bookworm defaults to LLVM 14.
RUN apt-get update && apt-get install -y \
    clang-15 \
    llvm-15 \
    make \
    git \
    liburing-dev \
    libmariadb-dev-compat \
    libsodium-dev \
    libcurl4-openssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Add LLVM 15 to PATH so `wolf build` finds the right toolchain
ENV PATH="/usr/lib/llvm-15/bin:${PATH}"

WORKDIR /app

# Copy project source
COPY . .

# Build the Wolf compiler CLI locally
RUN go build -o /usr/local/bin/wolf ./cmd/wolf

# Build the native bench_server binary
RUN wolf build examples/bench_server.wolf

# -----------------------------------------------------------------------------
# STAGE 2: Runner
# -----------------------------------------------------------------------------
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libcurl4 \
    liburing2 \
    libmariadb3 \
    libsodium23 \
    && rm -rf /var/lib/apt/lists/*

# Copy the compiled native binary from the builder stage
COPY --from=builder /app/wolf_out /app/wolf_out

# Expose standard Wolf API port (override if necessary)
EXPOSE 8080

# Run the compiled bench_server
CMD ["/app/wolf_out/bench_server"]
