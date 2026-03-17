# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' -o /out/caldo ./cmd/caldo

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/caldo /usr/local/bin/caldo
COPY configs/docker/config.docker.yaml /app/configs/docker/config.docker.yaml
COPY web/templates /app/web/templates
COPY web/static /app/web/static

ENV CALDO_CONFIG=/app/configs/docker/config.docker.yaml

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

CMD ["caldo"]
