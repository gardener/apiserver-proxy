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
	"net"
	"os"
	"strings"
	"time"

	utiliptables "github.com/gardener/apiserver-proxy/internal/iptables"
	"github.com/gardener/apiserver-proxy/internal/netif"
	"k8s.io/klog"
	"k8s.io/utils/exec"
)

// NewCacheApp returns a new instance of CacheApp by applying the specified config params.
func NewCacheApp(params *ConfigParams) (*CacheApp, error) {
	c := &CacheApp{params: params}
	if params.IPAddress == "" {
		klog.Infof("No ip-address provided, using environment variable")

		c.localIPStr = os.ExpandEnv("${KUBERNETES_SERVICE_HOST}")
	} else {
		klog.Infof("Using IP address %q", params.IPAddress)

		c.localIPStr = params.IPAddress
	}

	c.localIP = net.ParseIP(c.localIPStr)

	if c.localIP == nil {
		return nil, fmt.Errorf("unable to lookup IP address of Upstream service kubernetes, ip address was %q", c.localIPStr)
	}

	klog.Infof("Using %q as master service IP", c.localIPStr)

	return c, nil
}

// TeardownNetworking removes all custom iptables rules and network interface added by node-cache
func (c *CacheApp) TeardownNetworking() error {
	klog.Infof("Cleaning up")

	err := c.netManager.RemoveDummyDevice()

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

func (c *CacheApp) getIPTables() utiliptables.Interface {
	// using the localIPStr param since we need ip strings here
	c.iptablesRules = append(c.iptablesRules, []iptablesRule{
		// Match traffic destined for localIp:localPort and set the flows to be NOTRACKED, this skips connection tracking
		{utiliptables.Table("raw"), utiliptables.ChainPrerouting, []string{"-p", "tcp", "-d", c.localIPStr,
			"--dport", c.params.LocalPort, "-j", "NOTRACK"}},
		// There are rules in filter table to allow tracked connections to be accepted. Since we skipped connection tracking,
		// need these additional filter table rules.
		{utiliptables.TableFilter, utiliptables.ChainInput, []string{"-p", "tcp", "-d", c.localIPStr,
			"--dport", c.params.LocalPort, "-j", "ACCEPT"}},
		// Match traffic from c.localIPStr:localPort and set the flows to be NOTRACKED, this skips connection tracking
		{utiliptables.Table("raw"), utiliptables.ChainOutput, []string{"-p", "tcp", "-s", c.localIPStr,
			"--sport", c.params.LocalPort, "-j", "NOTRACK"}},
		// Additional filter table rules for traffic frpm c.localIPStr:localPort
		{utiliptables.TableFilter, utiliptables.ChainOutput, []string{"-p", "tcp", "-s", c.localIPStr,
			"--sport", c.params.LocalPort, "-j", "ACCEPT"}},
		// Skip connection tracking for requests to apiserver-proxy that are locally generated, example - by hostNetwork pods
		{utiliptables.Table("raw"), utiliptables.ChainOutput, []string{"-p", "tcp", "-d", c.localIPStr,
			"--dport", c.params.LocalPort, "-j", "NOTRACK"}},
	}...)
	execer := exec.New()

	return utiliptables.New(execer, utiliptables.ProtocolIpv4)
}

func (c *CacheApp) runPeriodic() {
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

func (c *CacheApp) runChecks() {
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

	klog.V(2).Infoln("Ensuring dummy interface")

	if err := c.netManager.EnsureDummyDevice(); err != nil {
		klog.Errorf("Error ensuring interface: %v", err)
	}

	klog.V(2).Infoln("Ensured dummy interface")
}

// RunApp invokes the background checks and runs coreDNS as a cache
func (c *CacheApp) RunApp(stopCh <-chan struct{}) {
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

	// Ensure that the required setup is ready
	// https://github.com/kubernetes/dns/issues/282
	// sometimes the interface gets the ip and then loses it, if added too soon.
	c.runChecks()
	// run periodic blocks
	c.runPeriodic()
}

func isLockedErr(err error) bool {
	return strings.Contains(err.Error(), "holding the xtables lock")
}
