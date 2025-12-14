# Multi-stage build for PAW Flask Explorer
FROM python:3.12-slim AS builder

LABEL maintainer="PAW Chain Team"
LABEL description="PAW Blockchain Explorer - Flask Application"

# Set build arguments
ARG FLASK_ENV=production

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    make \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy requirements first for better caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --upgrade pip setuptools wheel && \
    pip install --no-cache-dir -r requirements.txt

# Final stage
FROM python:3.12-slim

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    FLASK_APP=app.py \
    FLASK_ENV=production

# Create non-root user
RUN groupadd -r explorer && useradd -r -g explorer explorer

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy Python packages from builder
COPY --from=builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Copy application code
COPY --chown=explorer:explorer . .

# Create necessary directories
RUN mkdir -p /app/logs /app/static /app/templates && \
    chown -R explorer:explorer /app

# Switch to non-root user
USER explorer

# Expose port
EXPOSE 5000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:5000/health || exit 1

# Run application with gunicorn
CMD ["gunicorn", \
    "--bind", "0.0.0.0:5000", \
    "--workers", "4", \
    "--threads", "2", \
    "--worker-class", "gthread", \
    "--worker-tmp-dir", "/dev/shm", \
    "--timeout", "120", \
    "--keep-alive", "5", \
    "--log-level", "info", \
    "--access-logfile", "-", \
    "--error-logfile", "-", \
    "--access-logformat", "%(h)s %(l)s %(u)s %(t)s '%(r)s' %(s)s %(b)s '%(f)s' '%(a)s' %(D)s", \
    "app:app"]
