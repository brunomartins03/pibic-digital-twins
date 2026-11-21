# Relay server image: static Go binary on a minimal Alpine base.

FROM golang:1.24-alpine3.21 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/relay ./cmd/relay

FROM alpine:3.21
# ca-certificates for ssl:// and wss:// brokers; wget (busybox) for the healthcheck.
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 relay
COPY --from=build /out/relay /usr/local/bin/relay
USER relay
ENV RELAY_ADDR=:8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1
ENTRYPOINT ["relay"]
