#############      builder              #############
FROM golang:1.19.3 AS builder

WORKDIR /go/src/github.com/gardener/apiserver-proxy
COPY . .

ARG TARGETARCH
ARG EFFECTIVE_VERSION

RUN make build GOARCH=$TARGETARCH EFFECTIVE_VERSION=$EFFECTIVE_VERSION

#############      apiserver-proxy-builder      #############
FROM alpine:3.16.3 AS apiserver-proxy-builder

RUN apk add --no-cache iptables iproute2-minimal

WORKDIR /volume

COPY --from=builder /go/src/github.com/gardener/apiserver-proxy/bin/apiserver-proxy-sidecar ./apiserver-proxy-sidecar

RUN mkdir -p ./sbin ./lib ./usr/lib ./usr/lib/xtables ./tmp ./run ./etc/iproute2 \
    && cp -d /lib/ld-musl-* ./lib                                           && echo "package musl" \
    && cp -d /lib/libc.musl-* ./lib                                         && echo "package musl" \
    && cp -d /usr/lib/libcap.* ./usr/lib                                    && echo "package libcap" \
    && cp -d /usr/lib/libpsx.* ./usr/lib                                    && echo "package libcap" \
    && cp -d /usr/lib/libbz2* ./usr/lib                                     && echo "package libbz2" \
    && cp -d /usr/lib/libfts* ./usr/lib                                     && echo "package fts" \
    && cp -d /usr/lib/liblzma* ./usr/lib                                    && echo "package xz-libs" \
    && cp -d /lib/libz.* ./lib                                              && echo "package zlib" \
    && cp -d /usr/lib/libasm* ./usr/lib                                     && echo "package libelf" \
    && cp -d /usr/lib/libdw* ./usr/lib                                      && echo "package libelf" \
    && cp -d /usr/lib/libelf* ./usr/lib                                     && echo "package libelf" \
    && cp -d /usr/lib/libmnl.* ./usr/lib                                    && echo "package libmnl" \
    && cp -d /sbin/ip ./sbin                                                && echo "package iproute2-minimal" \
    && cp -d /etc/iproute2/* ./etc/iproute2                                 && echo "package iproute2-minimal" \
    && cp -d /usr/lib/libnftnl* ./usr/lib                                   && echo "package libnftnl" \
    && cp -d /etc/ethertypes ./etc                                          && echo "package iptables" \
    && cp -d /sbin/iptables* ./sbin                                         && echo "package iptables" \
    && cp -d /sbin/xtables* ./sbin                                          && echo "package iptables" \
    && cp -d /usr/lib/libip4* ./usr/lib                                     && echo "package iptables" \
    && cp -d /usr/lib/libip6* ./usr/lib                                     && echo "package iptables" \
    && cp -d /usr/lib/libipq* ./usr/lib                                     && echo "package iptables" \
    && cp -d /usr/lib/libxtables* ./usr/lib                                 && echo "package iptables" \
    && cp -d /usr/lib/xtables/* ./usr/lib/xtables                           && echo "package iptables"

#############      apiserver-proxy      #############
FROM scratch AS apiserver-proxy

WORKDIR /

COPY --from=apiserver-proxy-builder /volume /

USER 0:0

ENTRYPOINT ["/apiserver-proxy-sidecar"]

#############      apiserver-proxy-pod-webhook      #############
FROM gcr.io/distroless/static-debian11:nonroot AS apiserver-proxy-pod-webhook

COPY --from=builder /go/src/github.com/gardener/apiserver-proxy/bin/apiserver-proxy-pod-webhook /apiserver-proxy-pod-webhook
WORKDIR /

ENTRYPOINT ["/apiserver-proxy-pod-webhook"]
