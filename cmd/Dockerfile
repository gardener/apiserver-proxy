#############      builder              #############
FROM l.gcr.io/google/bazel:3.5.0 as builder-amd64
ARG ARCH=amd64

FROM l.gcr.io/google/bazel:3.5.0 as builder-arm64
ARG ARCH=arm64

FROM builder-$TARGETARCH as builder

WORKDIR /src/workspace
COPY . .

RUN bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_$ARCH "//cmd/apiserver-proxy-pod-webhook:apiserver-proxy-pod-webhook"
RUN bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_$ARCH "//cmd/apiserver-proxy-sidecar:apiserver-proxy-sidecar"

#############      apiserver-proxy-builder      #############
FROM alpine:3.15.4 AS apiserver-proxy-builder

RUN apk add --no-cache iptables iproute2

WORKDIR /volume

COPY --from=builder /src/workspace/bazel-bin/cmd/apiserver-proxy-sidecar/apiserver-proxy-sidecar_/apiserver-proxy-sidecar ./apiserver-proxy-sidecar

RUN mkdir -p ./sbin ./lib ./usr/lib \
    && cp -d /lib/ld-musl-* ./lib \
    && cp -d /lib/libc.musl-* ./lib \
    && cp -d /lib/libz.so.* ./lib \
    && cp -d /usr/lib/libelf* ./usr/lib \
    && cp -d /usr/lib/libip4* ./usr/lib \
    && cp -d /usr/lib/libip6* ./usr/lib \
    && cp -d /usr/lib/libxtables* ./usr/lib \
    && cp -d /usr/lib/libmnl* ./usr/lib \
    && cp -d /usr/lib/libnftnl* ./usr/lib \
    && cp -d /sbin/ip ./sbin \
    && cp -d /sbin/iptables* ./sbin \
    && cp -d /sbin/xtables* ./sbin

#############      apiserver-proxy      #############
FROM scratch AS apiserver-proxy

WORKDIR /

COPY --from=apiserver-proxy-builder /volume /

USER 0:0

ENTRYPOINT ["/apiserver-proxy-sidecar"]

#############      apiserver-proxy-pod-webhook      #############
FROM gcr.io/distroless/static-debian11:nonroot AS apiserver-proxy-pod-webhook

COPY --from=builder /src/workspace/bazel-bin/cmd/apiserver-proxy-pod-webhook/apiserver-proxy-pod-webhook_/apiserver-proxy-pod-webhook /apiserver-proxy-pod-webhook
WORKDIR /

ENTRYPOINT ["/apiserver-proxy-pod-webhook"]
