# syntax=docker/dockerfile:1

FROM golang:1.22 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/backend ./

FROM debian:bookworm-slim AS runner

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /bin/backend /usr/local/bin/backend

ENV PORT=8080
EXPOSE 8080

RUN useradd --system --home /app --shell /sbin/nologin backend
USER backend

ENTRYPOINT ["backend"]
