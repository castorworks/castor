## Build
FROM golang:1.26.8-bookworm AS build

WORKDIR /app

ARG GOPROXY=https://proxy.golang.org,direct
RUN go env -w GOPROXY=${GOPROXY}

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN cd cmd/castor && CGO_ENABLED=0 GOOS=linux go build -o /app/castor

## Runtime
FROM ubuntu:22.04 AS runtime

WORKDIR /app

# Containers default to UTC; override with --build-arg TZ=<zone> or the TZ environment variable
# (Go reads TZ with the zoneinfo from tzdata).
ARG TZ=UTC
ENV TZ=${TZ}
ARG UBUNTU_MIRROR=archive.ubuntu.com
ARG UBUNTU_SECURITY_MIRROR=security.ubuntu.com

RUN sed -i "s@//.*archive.ubuntu.com@//${UBUNTU_MIRROR}@g" /etc/apt/sources.list && \
    sed -i "s@security.ubuntu.com@${UBUNTU_SECURITY_MIRROR}@g" /etc/apt/sources.list && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl tzdata && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /app/castor /app/castor
COPY ./configs/i18n /app/configs/i18n

RUN groupadd -g 10001 castor && useradd -u 10001 -g castor -M castor && \
    mkdir -p /app/logs && chown -R castor:castor /app
USER 10001:10001

EXPOSE 1234

ENTRYPOINT ["/app/castor"]
