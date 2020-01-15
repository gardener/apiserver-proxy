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

	params := &app.ConfigParams{LocalPort: "443"}

	cmd := goflag.CommandLine
	klog.InitFlags(cmd)
	flag.CommandLine.AddGoFlagSet(cmd)

	flag.StringVar(&params.Interface, "interface", "apisrv0", "[optional] name of the interface to be created")
	flag.DurationVar(&params.Interval, "syncinterval", time.Minute, "[optional] interval to check for iptables rules")
	flag.BoolVar(&params.SetupIptables, "setupiptables", true, "indicates whether iptables rules should be setup")
	flag.BoolVar(&params.Cleanup, "cleanup", false,
		"indicates whether created interface and iptables should be removed on exit")
	flag.StringVar(&params.IPAddress, "ip-address", "", `[optional] ip-address on which the proxy is listening.
		If not set, it uses the "KUBERNETES_SERVICE_HOST" environment variable`)
	flag.Parse()

	return params
}

func main() {
	params := parseAndValidateFlags()

	cache, err := app.NewCacheApp(params)
	if err != nil {
		klog.Fatalf("Failed to obtain CacheApp instance, err %v", err)
	}

	cache.RunApp(signals.SetupSignalHandler())
}
