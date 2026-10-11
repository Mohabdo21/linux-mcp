FROM golang:1.27 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
        -mod=readonly -trimpath \
        -ldflags="-w -s -X main.Version=${VERSION}" \
        -o /app/linux-mcp .

FROM debian:trixie-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /app/linux-mcp /app/linux-mcp

WORKDIR /app

ENTRYPOINT ["/app/linux-mcp"]
