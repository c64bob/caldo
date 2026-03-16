# syntax=docker/dockerfile:1

FROM golang:1.22-alpine

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ENV CALDO_CONFIG=/app/configs/docker/config.docker.yaml

EXPOSE 8080

CMD ["go", "run", "./cmd/caldo"]
