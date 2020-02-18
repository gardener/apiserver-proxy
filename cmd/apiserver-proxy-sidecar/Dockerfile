#############      builder              #############
FROM l.gcr.io/google/bazel:2.1.0 as builder

WORKDIR /src/workspace
COPY . .
# COPY --from=tester /src/output /src/output

RUN bazel test --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 -- "//..." -//cmd/apiserver-proxy-sidecar/...
RUN bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 "//cmd/apiserver-proxy-sidecar:apiserver-proxy-sidecar"

#############      apiserver-proxy      #############
FROM alpine:3.11.3 AS apiserver-proxy

RUN apk add --no-cache iptables iproute2

COPY --from=builder /src/workspace/bazel-bin/cmd/apiserver-proxy-sidecar/linux_amd64_pure_stripped/apiserver-proxy-sidecar /apiserver-proxy-sidecar
WORKDIR /

USER 0:0

ENTRYPOINT ["/apiserver-proxy-sidecar"]
