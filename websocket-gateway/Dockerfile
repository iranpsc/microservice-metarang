# WebSocket Gateway Dockerfile
FROM node:18-alpine

# Set working directory
WORKDIR /app

# Copy package files
COPY ./websocket-gateway/package*.json ./

# Install dependencies
RUN npm install --only=production

# Copy application files
COPY ./websocket-gateway/ .

# Copy shared proto files for gRPC client
COPY ./shared/proto ./shared/proto

# Expose WebSocket port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3000/health', (r) => process.exit(r.statusCode === 200 ? 0 : 1))"

# Start server
CMD ["node", "server.js"]

