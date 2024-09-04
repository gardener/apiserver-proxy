// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/vishvananda/netlink"
	"golang.org/x/xerrors"
	"k8s.io/klog/v2"

	"github.com/gardener/apiserver-proxy/internal/netif"
)

// NewSidecarApp returns a new instance of SidecarApp by applying the specified config params.
func NewSidecarApp(params *ConfigParams) (*SidecarApp, error) {
	c := &SidecarApp{params: params}

	ip, err := netip.ParseAddr(c.params.IPAddress)
	if err != nil {
		return nil, xerrors.Errorf("unable to parse IP address %q - %v", c.params.IPAddress, err)
	}

	addr, err := netlink.ParseAddr(fmt.Sprintf("%s/%d", c.params.IPAddress, ip.BitLen()))
	if err != nil || addr == nil {
		return nil, xerrors.Errorf("unable to parse IP address %q - %v", c.params.IPAddress, err)
	}

	c.localIP = addr

	klog.Infof("Using IP address %q", params.IPAddress)

	return c, nil
}

// TeardownNetworking removes the network interface added by apiserver-proxy
func (c *SidecarApp) TeardownNetworking() error {
	klog.Infof("Cleaning up")
	err := c.netManager.RemoveIPAddress()
	if err != nil {
		return err
	}
	return c.netManager.CleanupDevice()
}

func (c *SidecarApp) runPeriodic(ctx context.Context) {
	tick := time.NewTicker(c.params.Interval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			klog.Warningf("Exiting interface check goroutine")

			return
		case <-tick.C:
			c.runChecks()
		}
	}
}

func (c *SidecarApp) runChecks() {

	klog.V(2).Infoln("Ensuring ip address")

	if err := c.netManager.EnsureIPAddress(); err != nil {
		klog.Errorf("Error ensuring ip address: %v", err)
	}

	klog.V(2).Infoln("Ensured ip address")
}

// RunApp invokes the background checks and runs coreDNS as a cache
func (c *SidecarApp) RunApp(ctx context.Context) {
	c.netManager = netif.NewNetifManager(c.localIP, c.params.Interface)

	if c.params.Cleanup {
		defer func() {
			if err := c.TeardownNetworking(); err != nil {
				klog.Fatalf("Failed to clean up - %v", err)
			}

			klog.Infoln("Successfully cleaned up everything. Bye!")
		}()
	}

	c.runChecks()

	if c.params.Daemon {
		klog.Infoln("Running as a daemon")
		// run periodic blocks
		c.runPeriodic(ctx)
	}

	klog.Infoln("Exiting... Bye!")
}
