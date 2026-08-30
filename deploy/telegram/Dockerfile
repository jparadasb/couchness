FROM golang:1.20.14-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/couchness .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1000 couchness \
    && adduser -D -u 1000 -G couchness couchness

COPY --from=builder /out/couchness /usr/local/bin/couchness
COPY deploy/telegram/entrypoint.sh /usr/local/bin/couchness-telegram-entrypoint

USER couchness
ENTRYPOINT ["/usr/local/bin/couchness-telegram-entrypoint"]
