// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package netif

import (
	"errors"
	"os"

	"github.com/vishvananda/netlink"
	"golang.org/x/xerrors"
	"k8s.io/klog/v2"
)

type Handle interface {
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	AddrDel(link netlink.Link, addr *netlink.Addr) error
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)
	LinkByName(name string) (netlink.Link, error)
	LinkSetUp(netlink.Link) error
	LinkAdd(netlink.Link) error
	LinkDel(netlink.Link) error
	LinkList() ([]netlink.Link, error)
}

// Manager ensures that the dummy device is created or removed.
type Manager interface {
	EnsureIPAddress() error
	RemoveIPAddress() error
	CleanupDevice() error
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
		var linkNotFoundErr netlink.LinkNotFoundError
		if !errors.As(err, &linkNotFoundErr) {
			return xerrors.Errorf("could not get interface %s:\n%v", m.devName, err)
		}

		dummyLink := &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name: m.devName,
			},
		}
		err = m.LinkAdd(dummyLink)
		if err != nil {
			return xerrors.Errorf("could not add dummy interface %s:\n%v", m.devName, err)
		}

		err = m.LinkSetUp(dummyLink)
		if err != nil {
			return xerrors.Errorf("could not set interface %s up:\n%v", m.devName, err)
		}

		l = dummyLink
	}

	err = m.deduplicateIPAddress()
	if err != nil {
		return xerrors.Errorf("could not deduplicate IP address:\n%v", err)
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

// deduplicateIPAddress removes duplicates of the given IP address on other devices
func (m *netifManagerDefault) deduplicateIPAddress() error {
	klog.V(4).Infof("Deduplicating address %q", m.addr.String())
	links, err := m.LinkList()
	if err != nil {
		return xerrors.Errorf("could not list interfaces: %v", err)
	}
	for _, l := range links {
		if l.Attrs().Name == m.devName {
			// skip own link
			continue
		}
		addrs, err := m.AddrList(l, 0)
		if err != nil {
			return xerrors.Errorf("could not list addresses for interface %s: %v", l.Attrs().Name, err)
		}
		for _, addr := range addrs {
			if addr.Equal(*m.addr) {
				klog.Infof("Found duplicate address %q on interface %q. Removing it.", m.addr.String(), l.Attrs().Name)
				if err := m.AddrDel(l, &addr); err != nil {
					return xerrors.Errorf("could not delete duplicate address %q from interface %q: %v", m.addr.String(), l.Attrs().Name, err)
				}
			}
		}
	}
	return nil
}

// RemoveIPAddress removes the IP address from the given interface
func (m *netifManagerDefault) RemoveIPAddress() error {
	klog.V(4).Infof("Getting interface %q", m.devName)

	l, err := m.LinkByName(m.devName)
	if err != nil {
		return xerrors.Errorf("could not get interface %s:\n%v", m.devName, err)
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

func (m *netifManagerDefault) CleanupDevice() error {
	link, err := netlink.LinkByName(m.devName)
	if err != nil {
		var linkNotFoundErr netlink.LinkNotFoundError
		if !errors.As(err, &linkNotFoundErr) {
			return xerrors.Errorf("could not get interface %s:\n%v", m.devName, err)
		}
		// link already gone
		return nil
	}
	err = netlink.LinkDel(link)
	if err != nil {
		return xerrors.Errorf("could not delete interface %s:\n%v", m.devName, err)
	}
	return nil
}
