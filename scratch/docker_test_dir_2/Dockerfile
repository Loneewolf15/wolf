# -----------------------------------------------------------------------------
# STAGE 1: Builder
# -----------------------------------------------------------------------------
FROM debian:bookworm-slim AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    curl \
    clang \
    llvm \
    make \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Wolf compiler (simulate curl install)
RUN curl -sL https://get.wolf-lang.org | bash
ENV PATH="/root/.wolf/bin:${PATH}"

# Set working directory
WORKDIR /app

# Copy project source
COPY . .

# Build the native binary
# The entry point defaults to public/index.wolf for APIs, src/main.wolf for scripts
RUN if [ -f "public/index.wolf" ]; then \
      wolf build public/index.wolf; \
    else \
      wolf build src/main.wolf; \
    fi

# -----------------------------------------------------------------------------
# STAGE 2: Runner
# -----------------------------------------------------------------------------
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies (e.g., ca-certificates for HTTPS, libcurl for wolf_http)
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libcurl4 \
    && rm -rf /var/lib/apt/lists/*

# Copy the compiled native binary from the builder stage
# Assuming the compiled binary takes the name of the project or defaults to the source name
# Here we just copy the entire wolf_out directory which contains the binary
COPY --from=builder /app/wolf_out /app/wolf_out

# Expose standard Wolf API port (override if necessary)
EXPOSE 2006

# Run the compiled binary (assuming the binary name is index if public/index.wolf was compiled)
CMD ["sh", "-c", "if [ -f /app/wolf_out/index ]; then /app/wolf_out/index; else /app/wolf_out/main; fi"]
