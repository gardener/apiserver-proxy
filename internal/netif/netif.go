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

// copied from golang.org/x/sys/unix constants
const (
	// copied from RT_TABLE_LOCAL
	// Also visible in /etc/iproute2/rt_tables file on linux hosts
	localRoutingTableId int = 0xff
	// copied from RT_SCOPE_HOST
	// Also visible in /etc/iproute2/rt_scopes file on linux hosts
	hostScopeId int = 0xfe
)

type Handle interface {
	RouteAdd(route *netlink.Route) error
	RouteDel(route *netlink.Route) error
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	AddrDel(link netlink.Link, addr *netlink.Addr) error
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
			return xerrors.Errorf("could add dummy interface %s:\n%v", m.devName, err)
		}

		err = m.LinkSetUp(dummyLink)
		if err != nil {
			return xerrors.Errorf("could set interface %s up:\n%v", m.devName, err)
		}

		l = dummyLink
	}

	klog.V(6).Infof("Got interface %+v", l)

	if err := m.AddrAdd(l, m.addr); err != nil {
		if !os.IsExist(err) {
			return xerrors.Errorf("could not add IPV4 addresses %v", err)
		}
		klog.V(4).Infof("Address %q already exists.", m.addr.String())
	}

	// loopback device adds new ip addresses to the local routing table by default.
	// If we are using a different interface, we need to do the updates ourself
	if l.Attrs().Name != "lo" {
		route := &netlink.Route{
			LinkIndex: l.Attrs().Index,
			Table:     localRoutingTableId,
			Dst:       m.addr.IPNet,
			Scope:     netlink.Scope(hostScopeId),
		}
		if err = m.RouteAdd(route); err != nil {
			if !os.IsExist(err) {
				return xerrors.Errorf("could not add route for %s to interface %s:\n%v", m.addr, m.devName, err)
			}
			klog.V(4).Infof("Route for %q already exists", m.addr.String())
		}
	}
	klog.Infof("Successfully added %q to %q", m.addr.String(), m.devName)

	return nil
}

// EnsureIPAddress makes sure to have the device running as desired.
func (m *netifManagerDefault) RemoveIPAddress() error {
	klog.V(4).Infof("Getting interface %q", m.devName)

	l, err := m.LinkByName(m.devName)
	if err != nil {
		return xerrors.Errorf("could not get interface %s:\n%v", m.devName, err)
	}

	klog.V(6).Infof("Got interface %+v", l)

	if m.devName != "lo" {
		route := localRoute(l.Attrs().Index, m.addr)
		if err := m.RouteDel(route); err != nil {
			if !os.IsNotExist(err) {
				return xerrors.Errorf("could not remove route for %s from interface %s:\n%v", m.addr, m.devName, err)
			}
			klog.V(4).Infof("Route for %q already removed. Skipping", m.addr.String())
		}
	}

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

func localRoute(linkIndex int, addr *netlink.Addr) *netlink.Route {
	return &netlink.Route{
		LinkIndex: linkIndex,
		Table:     localRoutingTableId,
		Dst:       addr.IPNet,
		Src:       addr.IP,
		Scope:     netlink.Scope(hostScopeId),
	}
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
