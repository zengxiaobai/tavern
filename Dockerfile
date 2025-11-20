# multi-stage Dockerfile for a Go app
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git ca-certificates

WORKDIR /src
# cache modules
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w" -o /app/app ./...

FROM alpine:latest

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache curl \
    && mkdir -p /usr/local/tavern/logs 

COPY bin/tavern /usr/local/tavern

WORKDIR /usr/local/tavern

EXPOSE 8080

ENV TZ=Asia/Shanghai

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
    CMD [ "curl", "-sS", "http://localhost:8080/healthz" ] || exit 1

ENTRYPOINT ["/usr/local/tavern/tavern", "-c", "/usr/local/tavern/config.yaml"]