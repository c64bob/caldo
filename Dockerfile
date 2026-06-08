FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN apk add --no-cache curl
RUN go install github.com/a-h/templ/cmd/templ@v0.3.865
ARG TAILWIND_VERSION=v3.4.17
ARG TAILWIND_SHA256=7d24f7fa191d2193b78cd5f5a42a6093e14409521908529f42d80b11fde1f1d4
RUN curl -sSL "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-x64" -o /usr/local/bin/tailwindcss \
    && echo "${TAILWIND_SHA256}  /usr/local/bin/tailwindcss" | sha256sum -c - \
    && chmod +x /usr/local/bin/tailwindcss
COPY . .
RUN templ generate
RUN ./scripts/build-assets.sh
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
