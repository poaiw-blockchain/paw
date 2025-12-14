# PAW Staking Dashboard
FROM nginx:alpine

# Install necessary packages
RUN apk add --no-cache curl

# Create app directory
WORKDIR /usr/share/nginx/html

# Copy dashboard files
COPY index.html .
COPY app.js .
COPY package.json .
COPY components/ ./components/
COPY services/ ./services/
COPY styles/ ./styles/
COPY utils/ ./utils/

# Create config injection script
RUN echo '#!/bin/sh' > /docker-entrypoint.d/40-inject-config.sh && \
    echo 'cat > /usr/share/nginx/html/config.js << EOF' >> /docker-entrypoint.d/40-inject-config.sh && \
    echo 'window.PAW_API_URL = "${PAW_API_URL}";' >> /docker-entrypoint.d/40-inject-config.sh && \
    echo 'window.PAW_RPC_URL = "${PAW_RPC_URL}";' >> /docker-entrypoint.d/40-inject-config.sh && \
    echo 'window.PAW_WS_URL = "${PAW_WS_URL}";' >> /docker-entrypoint.d/40-inject-config.sh && \
    echo 'EOF' >> /docker-entrypoint.d/40-inject-config.sh && \
    chmod +x /docker-entrypoint.d/40-inject-config.sh

# Configure nginx
RUN echo 'server {' > /etc/nginx/conf.d/default.conf && \
    echo '    listen 3000;' >> /etc/nginx/conf.d/default.conf && \
    echo '    server_name _;' >> /etc/nginx/conf.d/default.conf && \
    echo '    root /usr/share/nginx/html;' >> /etc/nginx/conf.d/default.conf && \
    echo '    index index.html;' >> /etc/nginx/conf.d/default.conf && \
    echo '    location / {' >> /etc/nginx/conf.d/default.conf && \
    echo '        try_files $uri $uri/ /index.html;' >> /etc/nginx/conf.d/default.conf && \
    echo '        add_header Cache-Control "no-cache, no-store, must-revalidate";' >> /etc/nginx/conf.d/default.conf && \
    echo '    }' >> /etc/nginx/conf.d/default.conf && \
    echo '    location ~* \.(js|css)$ {' >> /etc/nginx/conf.d/default.conf && \
    echo '        add_header Cache-Control "public, max-age=3600";' >> /etc/nginx/conf.d/default.conf && \
    echo '    }' >> /etc/nginx/conf.d/default.conf && \
    echo '}' >> /etc/nginx/conf.d/default.conf

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:3000/ || exit 1

EXPOSE 3000

# Use nginx's default entrypoint which will run our config injection script
