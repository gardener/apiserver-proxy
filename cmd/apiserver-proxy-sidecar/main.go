// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	goflag "flag"
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/gardener/apiserver-proxy/internal/app"
	"github.com/gardener/apiserver-proxy/internal/version"
)

func parseAndValidateFlags() *app.ConfigParams {
	params := &app.ConfigParams{}

	cmd := goflag.CommandLine
	klog.InitFlags(cmd)
	flag.CommandLine.AddGoFlagSet(cmd)

	flag.StringVar(&params.Interface, "interface", "lo", "[optional] name of the interface to add address to.")
	flag.DurationVar(&params.Interval, "sync-interval", time.Minute, "[optional] interval to check for the added interface.")
	flag.BoolVar(&params.Cleanup, "cleanup", false,
		"[optional] indicates whether created interface should be removed on exit.")
	flag.BoolVar(&params.Daemon, "daemon", true,
		"[optional] indicates if the sidecar should run as a daemon")
	flag.StringVar(&params.IPAddress, "ip-address", "", "ip-address on which the proxy is listening (e.g. 1.2.3.4).")
	flag.StringVar(&params.LocalPort, "port", "9443", "[optional] port on which the proxy is listening.")
	flag.BoolVar(&params.ManageInterface, "manage-interface", false, "[optional] indicates wether the interface should be created/deleted by the app")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s (%s):\n", os.Args[0], version.Version())
		flag.PrintDefaults()
	}

	flag.Parse()

	if params.IPAddress == "" {
		klog.Errorln("--ip-address is required")
		os.Exit(1)
	}

	return params
}

func main() {
	params := parseAndValidateFlags()

	app, err := app.NewSidecarApp(params)
	if err != nil {
		klog.Fatalf("Failed to create sidecar application, err %v", err)
	}

	app.RunApp(signals.SetupSignalHandler())
}
