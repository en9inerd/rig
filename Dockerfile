# ---------- Build ----------
FROM --platform=$BUILDPLATFORM golang:1.26.1-alpine AS builder

RUN apk update && apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
      -gcflags="all=-l -B" \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /rig \
      ./cmd/rig

# ---------- Runtime ----------
FROM alpine:3.21

RUN apk update && \
    apk add --no-cache \
        ca-certificates \
        tzdata \
        curl

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /rig /app/rig

RUN mkdir -p /data && chown app:app /data
VOLUME /data

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/rig"]
