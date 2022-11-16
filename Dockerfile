FROM --platform=$BUILDPLATFORM golang:1.19-alpine AS builder
WORKDIR /go/src/app
COPY . .
ARG TARGETARCH
ARG TARGETOS
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build

FROM alpine
COPY --from=builder /go/src/app/mikrotik-exporter /mikrotik-exporter
EXPOSE 9436
HEALTHCHECK --interval=1m --timeout=1s --retries=1 \
    CMD wget -q --spider http://localhost:9436/live || exit 1
ENTRYPOINT ["/mikrotik-exporter"]
