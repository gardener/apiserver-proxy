// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	goflag "flag"
	"fmt"
	"os"
	"time"

	"github.com/gardener/apiserver-proxy/internal/app"
	flag "github.com/spf13/pflag"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func parseAndValidateFlags() *app.ConfigParams {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	params := &app.ConfigParams{}

	cmd := goflag.CommandLine
	klog.InitFlags(cmd)
	flag.CommandLine.AddGoFlagSet(cmd)

	flag.StringVar(&params.Interface, "interface", "lo", "[optional] name of the interface to add address to.")
	flag.DurationVar(&params.Interval, "sync-interval", time.Minute, "[optional] interval to check for iptables rules.")
	flag.BoolVar(&params.SetupIptables, "setup-iptables", false,
		"[optional] indicates whether iptables rules should be setup.")
	flag.BoolVar(&params.Cleanup, "cleanup", false,
		"[optional] indicates whether created interface and iptables should be removed on exit.")
	flag.BoolVar(&params.Daemon, "daemon", true,
		"[optional] indicates if the sidecar should run as a daemon")
	flag.StringVar(&params.IPAddress, "ip-address", "", "ip-address on which the proxy is listening (e.g. 1.2.3.4).")
	flag.StringVar(&params.LocalPort, "port", "443", "[optional] port on which the proxy is listening.")
	flag.Parse()

	if params.IPAddress == "" {
		klog.Fatalf("--ip-address is required")
	}

	return params
}

func main() {
	params := parseAndValidateFlags()

	cache, err := app.NewSidecarApp(params)
	if err != nil {
		klog.Fatalf("Failed to create sidecar application, err %v", err)
	}

	cache.RunApp(signals.SetupSignalHandler())
}
