# ============================================
# Stage 1: Install dependencies
# ============================================

ARG NODE_VERSION=24-slim

FROM node:${NODE_VERSION} AS dependencies

WORKDIR /app

# Install bun to use bun.lock for dependency resolution
ARG BUN_VERSION=1.3.13
RUN npm install -g bun@${BUN_VERSION}

# Copy package-related files to leverage Docker cache
COPY package.json bun.lock* ./

# Install dependencies with frozen lockfile for reproducible builds
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --no-save --frozen-lockfile

# ============================================
# Stage 2: Build the Next.js application
# ============================================

FROM node:${NODE_VERSION} AS builder

WORKDIR /app

COPY --from=dependencies /app/node_modules ./node_modules
COPY . .

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Build-time env vars — override these with --build-arg or in compose.yml
ARG CASTOR_API_URL=http://castor-api:1234
ARG NEXT_PUBLIC_API_URL=
ARG NEXT_PUBLIC_SENTRY_DISABLED=true
ARG NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE=0.1

ENV CASTOR_API_URL=${CASTOR_API_URL}
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
ENV NEXT_PUBLIC_SENTRY_DISABLED=${NEXT_PUBLIC_SENTRY_DISABLED}
ENV NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE=${NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE}
ENV BUILD_STANDALONE=true

RUN npm run build

# ============================================
# Stage 3: Production runner
# ============================================

FROM node:${NODE_VERSION} AS runner

WORKDIR /app

# npm is only needed while building. Removing it from the runtime image keeps
# its package-manager dependency tree out of the attack surface.
RUN rm -rf /usr/local/lib/node_modules/npm /usr/local/bin/npm /usr/local/bin/npx

ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
ENV NEXT_TELEMETRY_DISABLED=1
ENV CASTOR_API_URL=http://castor-api:1234
ENV NEXT_PUBLIC_API_URL=
ENV NEXT_PUBLIC_SENTRY_DISABLED=true
ENV NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE=0.1

# Copy public assets
COPY --from=builder --chown=node:node /app/public ./public

# Create .next dir with correct permissions for prerender cache
RUN mkdir .next && chown node:node .next

# Copy standalone output and static files
COPY --from=builder --chown=node:node /app/.next/standalone ./
COPY --from=builder --chown=node:node /app/.next/static ./.next/static

# Run as non-root user
USER node

EXPOSE 3000

CMD ["node", "server.js"]
