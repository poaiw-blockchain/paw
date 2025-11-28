FROM openjdk:17-slim

# Metadata
LABEL maintainer="PAW Blockchain Team"
LABEL description="TLA+ Formal Verification Container for PAW Blockchain"
LABEL version="1.0"

# Install dependencies
RUN apt-get update && \
    apt-get install -y \
        wget \
        ca-certificates \
        bash \
        procps \
    && rm -rf /var/lib/apt/lists/*

# Create working directory
WORKDIR /formal

# Download TLA+ tools
ENV TLC_VERSION=v1.8.0
RUN wget -q https://example.com/tlaplus/tlaplus/releases/download/${TLC_VERSION}/tla2tools.jar \
    && mkdir -p /opt/tla-tools \
    && mv tla2tools.jar /opt/tla-tools/tla2tools.jar

# Set environment variables
ENV TLC_HOME=/opt/tla-tools
ENV JAVA_OPTS="-XX:+UseParallelGC -Xmx8g -Dtlc2.tool.fp.FPSet.impl=tlc2.tool.fp.OffHeapDiskFPSet"

# Copy specifications
COPY *.tla *.cfg ./
COPY verify.sh verify-all.sh validate_syntax.sh ./

# Make scripts executable
RUN chmod +x verify.sh verify-all.sh validate_syntax.sh

# Create results directory
RUN mkdir -p /formal/verification_results

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD java -version || exit 1

# Default command
ENTRYPOINT ["/formal/verify-all.sh"]
CMD []

# Usage:
# Build: docker build -t paw-formal-verification .
# Run all: docker run --rm paw-formal-verification
# Run specific: docker run --rm paw-formal-verification dex
# Quick mode: docker run --rm paw-formal-verification --quick
# Mount results: docker run --rm -v $(pwd)/results:/formal/verification_results paw-formal-verification
