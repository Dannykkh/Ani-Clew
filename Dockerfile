# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
COPY --from=frontend /app/web/dist ./internal/server/webdist/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /aniclew ./cmd/proxy/

# Stage 3: Final image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates git bash
COPY --from=builder /aniclew /usr/local/bin/aniclew
EXPOSE 4000
VOLUME /workspace
WORKDIR /workspace
ENTRYPOINT ["aniclew"]
CMD ["-port", "4000"]
