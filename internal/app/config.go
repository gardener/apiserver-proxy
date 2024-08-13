// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"time"

	"github.com/vishvananda/netlink"

	"github.com/gardener/apiserver-proxy/internal/netif"
)

// ConfigParams lists the configuration options that can be provided to sidecar proxy
type ConfigParams struct {
	// LocalPort specifies the port to listen for DNS requests
	LocalPort string
	// Interface specifies the name of the interface to be created
	Interface string
	// Interval specifies how often to run iptables rules check
	Interval time.Duration
	// SetupIptables enables iptables setup
	SetupIptables bool
	// Cleanup specifies whether to clean the created interface and iptables
	Cleanup bool
	// Daemon specifies whether to run as daemon
	Daemon bool
	// IPAddress specifies the IP address on which the proxy is listening
	IPAddress string
}

// SidecarApp contains all the config required to run sidecar proxy.
type SidecarApp struct {
	params     *ConfigParams
	netManager netif.Manager
	localIP    *netlink.Addr
}
