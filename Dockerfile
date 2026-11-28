# Multi-stage build for the Krill Billion Go server (Fly.io).
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates gzip
WORKDIR /app
COPY --from=build /out/server /app/server
COPY packs /app/packs
# Expand any committed *.json.gz packs for JSON fallback when Supabase is unset.
RUN for f in /app/packs/*.json.gz; do \
      [ -f "$f" ] || continue; \
      out="${f%.gz}"; \
      if [ ! -f "$out" ]; then gzip -dc "$f" > "$out"; fi; \
    done
ENV PACKS_DIR=/app/packs
ENV PORT=8080
EXPOSE 8080
USER nobody
CMD ["/app/server"]
