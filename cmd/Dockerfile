#############      builder              #############
FROM l.gcr.io/google/bazel:3.5.0 as builder

WORKDIR /src/workspace
COPY . .

RUN bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 "//cmd/apiserver-proxy-pod-webhook:apiserver-proxy-pod-webhook"
RUN bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 "//cmd/apiserver-proxy-sidecar:apiserver-proxy-sidecar"

#############      apiserver-proxy      #############
FROM alpine:3.15.3 AS apiserver-proxy

RUN apk add --no-cache iptables iproute2

COPY --from=builder /src/workspace/bazel-bin/cmd/apiserver-proxy-sidecar/apiserver-proxy-sidecar_/apiserver-proxy-sidecar /apiserver-proxy-sidecar
WORKDIR /

USER 0:0

ENTRYPOINT ["/apiserver-proxy-sidecar"]

#############      apiserver-proxy-pod-webhook      #############
FROM alpine:3.15.3 AS apiserver-proxy-pod-webhook

COPY --from=builder /src/workspace/bazel-bin/cmd/apiserver-proxy-pod-webhook/apiserver-proxy-pod-webhook_/apiserver-proxy-pod-webhook /apiserver-proxy-pod-webhook
WORKDIR /

USER 0:0

ENTRYPOINT ["/apiserver-proxy-pod-webhook"]
