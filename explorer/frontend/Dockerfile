# syntax=docker/dockerfile:1.4

ARG NODE_VERSION=18.20.5

FROM node:${NODE_VERSION}-alpine3.21 AS deps
WORKDIR /app
COPY package*.json ./
RUN \
  if [ -f package-lock.json ]; then npm ci; else npm install; fi

FROM node:${NODE_VERSION}-alpine3.21 AS builder
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:${NODE_VERSION}-alpine3.21 AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Copy standalone output
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000

CMD ["node", "server.js"]
