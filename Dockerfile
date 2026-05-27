# syntax=docker/dockerfile:1.7

# ---- Stage 1: build the Svelte SPA ----
FROM node:22-alpine AS spa
WORKDIR /spa
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci --no-audit --no-fund --loglevel=error
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: build the Go binary, embedding the SPA ----
FROM golang:1.23-alpine AS gobuild
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
# Inject the built SPA into the embed.FS path before compiling.
RUN rm -rf web/dist && mkdir -p web/dist
COPY --from=spa /spa/dist/ web/dist/

ENV CGO_ENABLED=0 GOOS=linux GOFLAGS=-trimpath
RUN go build -ldflags "-s -w" -o /out/rss-fresh ./cmd/rss-fresh

# ---- Stage 3: minimal runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
LABEL maintainer="Mustafa Eroğlu <mustafaeeroglu@icloud.com>" \
      org.opencontainers.image.source="https://github.com/MustafaEEroglu/rss-fresh" \
      org.opencontainers.image.description="Personal lightweight RSS / news manager"
WORKDIR /app
COPY --from=gobuild /out/rss-fresh /app/rss-fresh
USER nonroot:nonroot
EXPOSE 3000
ENTRYPOINT ["/app/rss-fresh"]
