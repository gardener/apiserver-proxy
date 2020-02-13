load("@io_bazel_rules_docker//container:container.bzl", "container_image")
load("@io_bazel_rules_docker//contrib:test.bzl", "container_test")
load("@io_bazel_rules_docker//docker/util:run.bzl", "container_run_and_commit")
load("@io_bazel_rules_docker//go:image.bzl", "go_image")
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

# Bootstrap the container image by installing some additional
# packages and running some post install steps
container_run_and_commit(
    name = "iptables_iproute_installer",
    commands = ["apk add --no-cache iptables iproute2 && rm -Rf /lib/apk/db/scripts.tar"],
    image = "@alpine_linux_amd64_tar//image:image.tar",
)

container_image(
    name = "iptables_iproute_installer_wrapper",
    base = ":iptables_iproute_installer_commit.tar",
    user = "65534:65534",
)

go_image(
    name = "go_image",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
    base = ":iptables_iproute_installer_wrapper",
)

container_test(
    name = "container_test",
    configs = ["//cmd/apiserver-proxy-sidecar/test_configs:installed_dependiencies.yaml"],
    image = ":go_image",
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/gardener/apiserver-proxy/cmd/apiserver-proxy-sidecar",
    visibility = ["//visibility:private"],
    deps = [
        "//internal/app:go_default_library",
        "@com_github_spf13_pflag//:go_default_library",
        "@io_k8s_klog//:go_default_library",
        "@io_k8s_sigs_controller_runtime//pkg/manager/signals:go_default_library",
    ],
)

go_binary(
    name = "apiserver-proxy-sidecar",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
