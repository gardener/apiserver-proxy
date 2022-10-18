# API Server proxy

This repository contains two components

- API Server proxy sidecar
- API Server proxy pod webhook

## API Server proxy sidecar

[![reuse compliant](https://reuse.software/badge/reuse-compliant.svg)](https://reuse.software/)

This component consists of `apiserver-proxy-sidecar` which runs on every `Node` in a `Shoot` cluster.
It does the following:

1. adds the IP Address (`--ip-address` flag) to the loopback interface  (`--interface` flag).
1. [optionally] executes the following `iptables` rules (e.g with `--ip-address=10.96.0.2`):

    ```text
    -A PREROUTING -t raw -d 10.96.0.2/32 -p tcp -m tcp --dport 443 -j NOTRACK
    -A OUTPUT -t raw -d 10.96.0.2/32 -p tcp -m tcp --dport 443 -j NOTRACK
    -A OUTPUT -t raw -s 10.96.0.2/32 -p tcp -m tcp --sport 443 -j NOTRACK
    -A INPUT -t filter -d 10.96.0.2/32 -p tcp -m tcp --dport 443 -j ACCEPT
    -A OUTPUT -t filter -s 10.96.0.2/32 -p tcp -m tcp --sport 443 -j ACCEPT
    ```

    Those rules allow traffic to this IP address and disable conntrack as the IP address is local to the machine.

1. Every 1 min repeats the process and starts from `1.`

After this, the actual `apiserver-proxy` can listen on this IP address (`10.96.0.2`) and send traffic to the correct kube-apiserver.
The implementation of that proxy is fully transparent and can be replaced at any given moment without any modifications to the `apiserver-proxy-sidecar`.

### Sidecar command line options

```console
go run ./cmd/apiserver-proxy-sidecar --help
      --add_dir_header                   If true, adds the file directory to the header
      --alsologtostderr                  log to standard error as well as files
      --cleanup                          [optional] indicates whether created interface and iptables should be removed on exit.
      --daemon                           [optional] indicates if the sidecar should run as a daemon (default true)
      --interface string                 [optional] name of the interface to add address to. (default "lo")
      --ip-address string                ip-address on which the proxy is listening.
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --log_file string                  If non-empty, use this log file
      --log_file_max_size uint           Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                      log to standard error instead of files (default true)
      --port string                      [optional] port on which the proxy is listening. (default "443")
      --setup-iptables                   [optional] indicates whether iptables rules should be setup.
      --skip_headers                     If true, avoid header prefixes in the log messages
      --skip_log_headers                 If true, avoid headers when opening log files
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
      --sync-interval duration           [optional] interval to check for iptables rules. (default 1m0s)
  -v, --v Level                          number for the log level verbosity
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

## API Server proxy pod webhook

The API Server proxy pod webhook server is a simple [mutating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) which adds the Fully Qualified Domain Name of the Kube API Server to Pods as environment variable `KUBERNETES_SERVICE_HOST`. The value is set with `--apiserver-fqdn` flag.

Test it with:

```console
go run ./cmd/apiserver-proxy-pod-webhook --apiserver-fqdn=foo.bar. --cert-dir ${PWD}/internal/admission/testdata
```

And in another terminal:

```console
curl -k -XPOST --silent \
 -H "Content-Type: application/json" \
 -d "@internal/admission/testdata/admission.json" \
 -H "Accept: application/json" \
https://localhost:9443/webhook/pod-apiserver-env | jq -r '.response.patch' | base64 -d | jq -r '.'
```

Output:

```json
[
  {
    "op": "add",
    "path": "/spec/initContainers/0/env",
    "value": [
      {
        "name": "KUBERNETES_SERVICE_HOST",
        "value": "foo.bar."
      }
    ]
  },
  {
    "op": "add",
    "path": "/spec/containers/0/env",
    "value": [
      {
        "name": "KUBERNETES_SERVICE_HOST",
        "value": "foo.bar."
      }
    ]
  }
]
```

### Webhook command line options

```console
go run ./cmd/apiserver-proxy-pod-webhook --help
      --apiserver-fqdn string   apiserver-fqdn is the fully qualified domain name of the Kube-API Server e.g. example.com.
      --cert-dir string         cert-dir is the directory that contains the server key and certificate. The server key and certificate.
      --cert-name string        [optional] cert-name is the server certificate name. (default "tls.crt")
      --client-ca-name string   [optional] client-ca-name is the CA certificate name which server used to verify remote(client)'s certificate. Defaults to "", which means server does not verify client's certificate.
      --host string             [optional] host is the address that the server will listen on. Defaults to "" - all addresses.
      --key-name string         [optional] key-name is the server key name. (default "tls.key")
      --port int                [optional] port is the port number that the server will serve. (default 9443)
```

## Development

### Update dependencies

```shell
make revendor
```

### Building container images

```shell
make docker-images
```

### Testing

```shell
make test
```
