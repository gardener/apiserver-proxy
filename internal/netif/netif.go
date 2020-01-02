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
	"net"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"golang.org/x/xerrors"
	"k8s.io/klog"
)

type Handle interface {
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	AddrDel(link netlink.Link, addr *netlink.Addr) error
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)
	LinkAdd(link netlink.Link) error
	LinkByName(name string) (netlink.Link, error)
	LinkDel(link netlink.Link) error
}

// Manager ensures that the dummy device is created or removed.
type Manager interface {
	EnsureDummyDevice() error
	RemoveDummyDevice() error
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
func NewNetifManager(ip net.IP, devName string) Manager {
	return &netifManagerDefault{
		&netlink.Handle{},
		&netlink.Addr{
			IPNet: netlink.NewIPNet(ip),
			// We only want the scope to be host
			Scope: 0xfe,
		},
		devName,
	}
}

// RemoveDummyDevice deletes the dummy device with the given name.
func (m *netifManagerDefault) RemoveDummyDevice() error {
	link, err := m.LinkByName(m.devName)
	if err != nil {
		return err
	}

	return m.LinkDel(link)
}

// EnsureDummyDevice makes sure to have the device running as desired.
// It'll remove unwanted ip addresses and ensure that the device
// is created of type "dummy".
func (m *netifManagerDefault) EnsureDummyDevice() error {
	dummy := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{Name: m.devName},
	}

	klog.V(4).Infof("Getting interface %q", m.devName)

	l, err := m.LinkByName(m.devName)
	klog.V(6).Infof("Got interface %+v", l)

	if err != nil {
		klog.Infof("Got error getting interface %+v", err)

		if isNotFoundError(err) {
			return m.addLink(dummy)
		}

		return xerrors.Errorf("could not get interface %w", err)
	}

	if l.Type() != "dummy" {
		klog.Warningf(`Interface is not of type "dummy" - %q. Deleting`, l.Type())

		if err := m.LinkDel(l); err != nil && !isNotFoundError(err) {
			return xerrors.Errorf("could not remove interface: %w", err)
		}

		return m.addLink(dummy)
	}

	klog.V(6).Infoln("Listing addresses on interface")

	aa, err := m.AddrList(l, unix.AF_INET)
	if err != nil {
		return xerrors.Errorf("could not list IPV4 addresses %w", err)
	}

	klog.V(6).Infof("Got addresses: %+v", aa)

	return m.ensureAddresses(l, aa)
}

func (m *netifManagerDefault) addAddress(l netlink.Link) error {
	klog.V(2).Infoln("Adding address")

	if err := m.AddrAdd(l, m.addr); err != nil {
		return xerrors.Errorf("could not add address %w", err)
	}

	return nil
}

func (m *netifManagerDefault) addLink(l netlink.Link) error {
	klog.V(2).Infoln("Interface not found. Creating")

	if err := m.LinkAdd(l); err != nil {
		return xerrors.Errorf("could not create interface %w", err)
	}

	return m.addAddress(l)
}

func (m *netifManagerDefault) ensureAddresses(l netlink.Link, aa []netlink.Addr) error {
	found := false

	for _, a := range aa {
		a := a
		if a.IPNet.String() == m.addr.IPNet.String() {
			klog.V(4).Infof("Address %q already exists. Skipping", m.addr.IPNet.String())

			found = true

			continue
		}

		klog.Warningf("Removing extra address %q", a.String())

		if err := m.AddrDel(l, &a); err != nil {
			return xerrors.Errorf("could not remove IPV4 address %w", err)
		}
	}

	if !found {
		return m.addAddress(l)
	}

	return nil
}

func isNotFoundError(err error) bool {
	_, got := err.(netlink.LinkNotFoundError)
	return got
}
