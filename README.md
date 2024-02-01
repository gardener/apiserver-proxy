# API Server proxy

This repository contains the API Server proxy sidecar.

The API Server proxy pod webhook is now included in the [Gardener Resource Manager](https://github.com/gardener/gardener/blob/master/docs/concepts/resource-manager.md#kubernetes-service-host-injection).

## API Server proxy sidecar

[![reuse compliant](https://reuse.software/badge/reuse-compliant.svg)](https://reuse.software/)

This component consists of `apiserver-proxy-sidecar` which runs on every `Node` in a `Shoot` cluster.
It does the following:

1. adds the IP Address (`--ip-address` flag) to the loopback interface  (`--interface` flag).

1. Every 1 min repeats the process and starts from `1.`

After this, the actual `apiserver-proxy` can listen on this IP address (`10.96.0.2`) and send traffic to the correct kube-apiserver.
The implementation of that proxy is fully transparent and can be replaced at any given moment without any modifications to the `apiserver-proxy-sidecar`.

### Sidecar command line options

```console
go run ./cmd/apiserver-proxy-sidecar --help
      --add_dir_header                   If true, adds the file directory to the header
      --alsologtostderr                  log to standard error as well as files
      --cleanup                          [optional] indicates whether created interface should be removed on exit.
      --daemon                           [optional] indicates if the sidecar should run as a daemon (default true)
      --interface string                 [optional] name of the interface to add address to. (default "lo")
      --ip-address string                ip-address on which the proxy is listening.
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --log_file string                  If non-empty, use this log file
      --log_file_max_size uint           Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                      log to standard error instead of files (default true)
      --port string                      [optional] port on which the proxy is listening. (default "443")
      --skip_headers                     If true, avoid header prefixes in the log messages
      --skip_log_headers                 If true, avoid headers when opening log files
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
      --sync-interval duration           [optional] interval to check for the added interface. (default 1m0s)
  -v, --v Level                          number for the log level verbosity
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

## Development

### Update dependencies

```shell
make tidy
```

### Building container images

```shell
make docker-images
```

### Testing

```shell
make test
```
