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
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /rig /app/rig
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER nonroot:nonroot

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD ["/app/rig", "--healthcheck"]

ENTRYPOINT ["/app/rig"]
