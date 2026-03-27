// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"time"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/gardener/apiserver-proxy/internal/netif"
)

var IPScopes map[string]int

func init() {
	IPScopes = make(map[string]int)
	IPScopes["global"] = unix.RT_SCOPE_UNIVERSE
	IPScopes["site"] = unix.RT_SCOPE_SITE
	IPScopes["host"] = unix.RT_SCOPE_HOST
	IPScopes["link"] = unix.RT_SCOPE_LINK
	IPScopes["nowhere"] = unix.RT_SCOPE_NOWHERE
}

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
	// IPScope specifies the scope of the IP address, see IPScopes map for supported values
	IPScope string
}

// SidecarApp contains all the config required to run sidecar proxy.
type SidecarApp struct {
	params     *ConfigParams
	netManager netif.Manager
	localIP    *netlink.Addr
}
