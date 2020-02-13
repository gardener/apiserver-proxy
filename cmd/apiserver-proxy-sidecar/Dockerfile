#############      builder              #############
FROM golang:1.13.7 AS builder

WORKDIR /go/src/github.com/gardener/apiserver-proxy-sidecar
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  \
  -o apiserver-proxy-sidecar \
  -mod=vendor \
  github.com/gardener/apiserver-proxy/cmd/apiserver-proxy-sidecar

#############      apiserver-proxy      #############
FROM alpine:3.11.3 AS apiserver-proxy

RUN apk add --no-cache iptables iproute2

COPY --from=builder /go/src/github.com/gardener/apiserver-proxy/apiserver-proxy-sidecar /apiserver-proxy-sidecar
WORKDIR /

USER 65534:65534

ENTRYPOINT ["/apiserver-proxy-sidecar"]
