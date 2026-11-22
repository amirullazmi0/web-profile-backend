FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

# ca-certificates is required to reach SMTP/TLS services; tzdata lets the
# container resolve a non-UTC DB_TIMEZONE.
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 app

WORKDIR /app
COPY --from=build /out/api ./api
COPY --from=build /src/templates ./templates

ENV APP_PORT=5101
EXPOSE 5101

# Swarm, which CapRover runs on, waits for this before shifting traffic to a new
# task. It checks liveness, not readiness, so a brief database outage does not
# turn into a restart loop.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/api", "-healthcheck"]

USER app
ENTRYPOINT ["/app/api"]
