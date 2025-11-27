FROM golang:1.24.6-bullseye AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod tidy

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o app .

FROM debian:12.12-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    useradd -r -u 1001 appuser

WORKDIR /app

COPY --from=builder /app/app .

USER appuser

ENTRYPOINT ["./app"]
