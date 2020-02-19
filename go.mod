module github.com/gardener/apiserver-proxy

go 1.13

require (
	github.com/golang/mock v1.3.1
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/spf13/pflag v1.0.3
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20190625233234-7109fa855b0f // indirect
	golang.org/x/sys v0.0.0-20191120155948-bd437916bb0e
	golang.org/x/xerrors v0.0.0-20190717185122-a985d3407aa7
	k8s.io/apimachinery v0.0.0-20191025225532-af6325b3a843 // kubernetes-1.18.0-alpha.0
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20191030222137-2b95a09bc58d
	sigs.k8s.io/controller-runtime v0.3.0
)
