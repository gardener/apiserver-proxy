// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"time"

	"github.com/vishvananda/netlink"

	utiliptables "github.com/gardener/apiserver-proxy/internal/iptables"
	"github.com/gardener/apiserver-proxy/internal/netif"
)

// ConfigParams lists the configuration options that can be provided to sidecar proxy
type ConfigParams struct {
	LocalPort     string        // port to listen for dns requests
	Interface     string        // Name of the interface to be created
	Interval      time.Duration // specifies how often to run iptables rules check
	SetupIptables bool          // enable iptables setup
	Cleanup       bool          // clean the created interface and iptables
	Daemon        bool          // run as a daemon
	IPAddress     string        // IP address on which the proxy is listening
}

// SidecarApp contains all the config required to run sidecar proxy.
type SidecarApp struct {
	iptables      utiliptables.Interface
	iptablesRules []iptablesRule
	params        *ConfigParams
	netManager    netif.Manager
	localIP       *netlink.Addr
}

type iptablesRule struct {
	table utiliptables.Table
	chain utiliptables.Chain
	args  []string
}
