// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/gardener/apiserver-proxy/internal/admission"
	"github.com/gardener/apiserver-proxy/internal/version"
)

var setupLog = ctrl.Log.WithName("setup")

func parseAndValidateFlags() (*webhook.Server, string) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	var (
		server        = &webhook.Server{}
		apiserverFQDN string
	)

	flag.StringVar(&apiserverFQDN, "apiserver-fqdn", "", `apiserver-fqdn is the fully qualified domain name of the Kube-API Server e.g. example.com.`)
	flag.StringVar(&server.CertDir, "cert-dir", "", "cert-dir is the directory that contains the server key and certificate. The server key and certificate.")
	flag.StringVar(&server.CertName, "cert-name", "tls.crt", "[optional] cert-name is the server certificate name.")
	flag.StringVar(&server.KeyName, "key-name", "tls.key", `[optional] key-name is the server key name.`)
	flag.StringVar(&server.ClientCAName, "client-ca-name", "", `[optional] client-ca-name is the CA certificate name which server used to verify remote(client)'s certificate. Defaults to "", which means server does not verify client's certificate.`)
	flag.StringVar(&server.Host, "host", "", `[optional] host is the address that the server will listen on. Defaults to "" - all addresses.`)
	flag.IntVar(&server.Port, "port", 9443, `[optional] port is the port number that the server will serve.`)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s (%s):\n", os.Args[0], version.Version())
		flag.PrintDefaults()
	}

	flag.Parse()

	if errs := validation.IsFullyQualifiedDomainName(&field.Path{}, apiserverFQDN); len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "--apiserver-fqdn %q is not a FQDN: %v\n", apiserverFQDN, errs.ToAggregate())
		os.Exit(1)
	}

	if server.CertDir == "" {
		fmt.Fprintf(os.Stderr, "--cert-dir is required\n")
		os.Exit(1)
	}

	server.CertDir = filepath.Clean(server.CertDir)

	return server, apiserverFQDN
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{})))

	srv, fqdn := parseAndValidateFlags()

	if err := admission.Complete(srv, fqdn); err != nil {
		setupLog.Error(err, "could not create manager")
		os.Exit(1)
	}

	if err := srv.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "could not start webhook server")
		os.Exit(1)
	}
}
