# Multi-stage build for the Krill Billion Go server (Fly.io).
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/server /app/server
COPY packs /app/packs
ENV PACKS_DIR=/app/packs
ENV PORT=8080
EXPOSE 8080
USER nobody
CMD ["/app/server"]
