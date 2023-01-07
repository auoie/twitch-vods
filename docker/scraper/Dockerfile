FROM golang:1.19 AS builder
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . ./
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -buildmode=pie -tags 'osusergo,netgo,static' -ldflags='-w -s -linkmode external -extldflags "-static-pie"' -o /twitch-vods-scraper ./cmd/testingScraper

FROM scratch
COPY --from=builder /twitch-vods-scraper /twitch-vods-scraper
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/twitch-vods-scraper"]
