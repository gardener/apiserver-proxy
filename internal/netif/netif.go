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

package netif

import (
	"os"

	"github.com/vishvananda/netlink"
	"golang.org/x/xerrors"
	"k8s.io/klog"
)

type Handle interface {
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	AddrDel(link netlink.Link, addr *netlink.Addr) error
	LinkByName(name string) (netlink.Link, error)
}

// Manager ensures that the dummy device is created or removed.
type Manager interface {
	EnsureIPAddress() error
	RemoveIPAddress() error
}

// netifManagerDefault is the default implementation handling creating
// and removing of the dummy interface.
type netifManagerDefault struct {
	Handle
	addr    *netlink.Addr
	devName string
}

// NewNetifManager returns a new instance of NetifManager with the ip address set to the provided values
// These ip addresses will be bound to any devices created by this instance.
func NewNetifManager(addr *netlink.Addr, devName string) Manager {
	// Set scope to host only
	addr.Scope = 0xfe

	return &netifManagerDefault{
		&netlink.Handle{},
		addr,
		devName,
	}
}

// EnsureIPAddress makes sure to have the device running as desired.
func (m *netifManagerDefault) EnsureIPAddress() error {
	klog.V(4).Infof("Getting interface %q", m.devName)

	l, err := m.LinkByName(m.devName)
	if err != nil {
		return xerrors.Errorf("could not get loopback interface %v", err)
	}

	klog.V(6).Infof("Got interface %+v", l)

	if err := m.AddrAdd(l, m.addr); err != nil {
		if os.IsExist(err) {
			klog.V(4).Infof("Address %q already exists. Skipping", m.addr.String())
			return nil
		}

		return xerrors.Errorf("could not add IPV4 addresses %v", err)
	}

	klog.Infof("Successfully added %q to %q", m.addr.String(), m.devName)

	return nil
}

// EnsureIPAddress makes sure to have the device running as desired.
func (m *netifManagerDefault) RemoveIPAddress() error {
	klog.V(4).Infof("Getting interface %q", m.devName)

	l, err := m.LinkByName(m.devName)
	if err != nil {
		return xerrors.Errorf("could not get loopback interface %v", err)
	}

	klog.V(6).Infof("Got interface %+v", l)

	if err := m.AddrDel(l, m.addr); err != nil {
		if os.IsNotExist(err) {
			klog.V(4).Infof("Address %q already removed. Skipping", m.addr.String())
			return nil
		}

		return xerrors.Errorf("could not delete ip address %v", err)
	}

	klog.Infof("Successfully removed %q from %q", m.addr.String(), m.devName)

	return nil
}
