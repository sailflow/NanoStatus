# Build stage for frontend
FROM oven/bun:1-slim AS frontend-builder
WORKDIR /app
COPY src/package.json src/bun.lock* ./
# Install all dependencies (including devDependencies needed for build)
RUN bun install --frozen-lockfile
COPY src/ ./src/
WORKDIR /app/src
RUN bun run build.ts --outdir=../dist && \
    find ../dist -type f -name "*.map" -delete && \
    find ../dist -type d -empty -delete

# Build stage for Go backend
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY *.go ./
COPY --from=frontend-builder /app/dist ./dist
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -trimpath -o nanostatus .

# Final stage - distroless static (no CGO needed)
FROM gcr.io/distroless/static:nonroot
COPY --from=backend-builder --chown=nonroot:nonroot /app/nanostatus /nanostatus
ENV PORT=8080 DB_PATH=/data/nanostatus.db ZEROLOG_LOG_LEVEL=info
EXPOSE 8080
VOLUME ["/data"]
USER nonroot:nonroot
ENTRYPOINT ["/nanostatus"]

