FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/caldo ./cmd/caldo

FROM alpine:3.22
WORKDIR /app
RUN adduser -D -u 1000 caldo && apk add --no-cache wget
COPY --from=builder /out/caldo /app/caldo
COPY web/static /app/web/static
USER caldo
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/app/caldo"]
