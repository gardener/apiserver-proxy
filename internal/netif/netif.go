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
		if _, ok := errors.AsType[netlink.LinkNotFoundError](err); !ok {
			return xerrors.Errorf("could not get interface %s:\n%v", m.devName, err)
		}

		dummyLink := &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name: m.devName,
			},
		}
		err = m.LinkAdd(dummyLink)
		if err != nil {
			return xerrors.Errorf("could add dummy interface %s:\n%v", m.devName, err)
		}

		err = m.LinkSetUp(dummyLink)
		if err != nil {
			return xerrors.Errorf("could set interface %s up:\n%v", m.devName, err)
		}

		l = dummyLink
	}

	klog.V(6).Infof("Got interface %+v", l)

	// Check if ip already exists on the link
	addrs, err := m.AddrList(l, netlink.FAMILY_V4)
	if err != nil {
		return xerrors.Errorf("could list addresses for interface %s:\n%v", m.devName, err)
	}

	for _, a := range addrs {
		if a.Equal(*m.addr) {
			if a.Scope == m.addr.Scope {
				klog.V(4).Infof("Address %q already exists. Skipping", m.addr.String())
				return nil
			}
			klog.V(4).Infof("Address %q scope mismatch. Deleting", m.addr.String())
			if err := m.AddrDel(l, &a); err != nil {
				return xerrors.Errorf("could not delete ip address with wrong scope %v", err)
			}
			break
		}
	}
	if err := m.AddrAdd(l, m.addr); err != nil {
		return xerrors.Errorf("could not add IPV4 address %v", err)
	}

	klog.Infof("Successfully added %q to %q", m.addr.String(), m.devName)

	return nil
}

// RemoveIPAddress removes the ip address again from the device. It does not remove the device itself.
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

// CleanupDevice removes the network device again.
func (m *netifManagerDefault) CleanupDevice() error {
	link, err := netlink.LinkByName(m.devName)
	if err != nil {
		if _, ok := errors.AsType[netlink.LinkNotFoundError](err); !ok {
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
