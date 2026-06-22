# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /app
# Copy module manifests first so `go mod download` is cached as its own layer
# and only re-runs when dependencies change, not on every source edit.
COPY go.mod go.sum ./
# Use the China-friendly Go proxy so module download doesn't time out inside
# the WeChat Cloud Run builder (which runs in mainland China).
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /server /app/server
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/server"]
