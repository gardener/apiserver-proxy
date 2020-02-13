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

package app

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/xerrors"

	utiliptables "github.com/gardener/apiserver-proxy/internal/iptables"
	"github.com/gardener/apiserver-proxy/internal/netif"
	"github.com/vishvananda/netlink"
	"k8s.io/klog"
	"k8s.io/utils/exec"
)

// NewSidecarApp returns a new instance of SidecarApp by applying the specified config params.
func NewSidecarApp(params *ConfigParams) (*SidecarApp, error) {
	c := &SidecarApp{params: params}

	addr, err := netlink.ParseAddr(fmt.Sprintf("%s/32", c.params.IPAddress))
	if err != nil || addr == nil {
		return nil, xerrors.Errorf("unable to parse IP address %q - %v", c.params.IPAddress, err)
	}

	c.localIP = addr

	klog.Infof("Using IP address %q", params.IPAddress)

	return c, nil
}

// TeardownNetworking removes all custom iptables rules and network interface added by node-cache
func (c *SidecarApp) TeardownNetworking() error {
	klog.Infof("Cleaning up")

	err := c.netManager.RemoveIPAddress()

	if c.params.SetupIptables {
		for _, rule := range c.iptablesRules {
			exists := true
			for exists {
				c.iptables.DeleteRule(rule.table, rule.chain, rule.args...)
				exists, _ = c.iptables.EnsureRule(utiliptables.Prepend, rule.table, rule.chain, rule.args...)
			}
			// Delete the rule one last time since EnsureRule creates the rule if it doesn't exist
			c.iptables.DeleteRule(rule.table, rule.chain, rule.args...)
		}
	}

	return err
}

func (c *SidecarApp) getIPTables() utiliptables.Interface {
	// using the localIPStr param since we need ip strings here
	c.iptablesRules = append(c.iptablesRules, []iptablesRule{
		// Match traffic destined for localIp:localPort and set the flows to be NOTRACKED, this skips connection tracking
		{utiliptables.Table("raw"), utiliptables.ChainPrerouting, []string{"-p", "tcp", "-d", c.params.IPAddress,
			"--dport", c.params.LocalPort, "-j", "NOTRACK"}},
		// There are rules in filter table to allow tracked connections to be accepted. Since we skipped connection tracking,
		// need these additional filter table rules.
		{utiliptables.TableFilter, utiliptables.ChainInput, []string{"-p", "tcp", "-d", c.params.IPAddress,
			"--dport", c.params.LocalPort, "-j", "ACCEPT"}},
		// Match traffic from c.params.IPAddress:localPort and set the flows to be NOTRACKED, this skips connection tracking
		{utiliptables.Table("raw"), utiliptables.ChainOutput, []string{"-p", "tcp", "-s", c.params.IPAddress,
			"--sport", c.params.LocalPort, "-j", "NOTRACK"}},
		// Additional filter table rules for traffic frpm c.params.IPAddress:localPort
		{utiliptables.TableFilter, utiliptables.ChainOutput, []string{"-p", "tcp", "-s", c.params.IPAddress,
			"--sport", c.params.LocalPort, "-j", "ACCEPT"}},
		// Skip connection tracking for requests to apiserver-proxy that are locally generated, example - by hostNetwork pods
		{utiliptables.Table("raw"), utiliptables.ChainOutput, []string{"-p", "tcp", "-d", c.params.IPAddress,
			"--dport", c.params.LocalPort, "-j", "NOTRACK"}},
	}...)
	execer := exec.New()

	return utiliptables.New(execer, utiliptables.ProtocolIpv4)
}

func (c *SidecarApp) runPeriodic() {
	tick := time.NewTicker(c.params.Interval)

	for {
		select {
		case <-tick.C:
			c.runChecks()
		case <-c.exitChan:
			klog.Warningf("Exiting iptables/interface check goroutine")
			return
		}
	}
}

func (c *SidecarApp) runChecks() {
	if c.params.SetupIptables {
		for _, rule := range c.iptablesRules {
			exists, err := c.iptables.EnsureRule(utiliptables.Prepend, rule.table, rule.chain, rule.args...)

			switch {
			case exists:
				// debug messages can be printed by including "debug" plugin in coreFile.
				klog.V(2).Infof("iptables rule %v for apiserver-proxy-sidecar already exists", rule)
				continue
			case err == nil:
				klog.Infof("Added back apiserver-proxy-sidecar rule - %v", rule)
				continue
			case isLockedErr(err):
				// if we got here, either iptables check failed or adding rule back failed.
				klog.Infof("Error checking/adding iptables rule %v, due to xtables lock in use, retrying in %v",
					rule, c.params.Interval)
			default:
				klog.Errorf("Error adding iptables rule %v - %s", rule, err)
			}
		}
	}

	klog.V(2).Infoln("Ensuring ip address")

	if err := c.netManager.EnsureIPAddress(); err != nil {
		klog.Errorf("Error ensuring ip address: %v", err)
	}

	klog.V(2).Infoln("Ensured ip address")
}

// RunApp invokes the background checks and runs coreDNS as a cache
func (c *SidecarApp) RunApp(stopCh <-chan struct{}) {
	c.netManager = netif.NewNetifManager(c.localIP, c.params.Interface)
	c.exitChan = stopCh

	if c.params.SetupIptables {
		c.iptables = c.getIPTables()
	}

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
		c.runPeriodic()
	}

	klog.Infoln("Exiting... Bye!")
}

func isLockedErr(err error) bool {
	return strings.Contains(err.Error(), "holding the xtables lock")
}
