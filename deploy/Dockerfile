FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /personal-agent ./cmd/personal-agent

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app \
    && mkdir -p /data \
    && chown app:app /data
USER app
WORKDIR /app
COPY --from=build /personal-agent /usr/local/bin/personal-agent
COPY --chown=app:app web /app/web
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1
CMD ["personal-agent"]
