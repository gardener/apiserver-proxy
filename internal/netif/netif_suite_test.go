// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package netif

import (
	"fmt"
	"net"
	"syscall"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	gomock "go.uber.org/mock/gomock"
)

func TestNetif(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Netif Suite")
}

var _ = Describe("Manager", func() {

	var (
		ctrl            *gomock.Controller
		mh              *MockHandle
		addr            *netlink.Addr
		interfaceName   string
		manageInterface bool
		manager         Manager
		dm              *netifManagerDefault
		dummy           *netlink.Dummy
		ip              = "192.168.0.3"
	)

	BeforeEach(func() {
		addr, _ = netlink.ParseAddr(ip + "/32")
		interfaceName = "foo"
		manageInterface = false
		ctrl = gomock.NewController(GinkgoT())
		mh = NewMockHandle(ctrl)
		dummy = &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{Name: interfaceName},
		}
	})

	AfterEach(func() {

		mh.EXPECT().AddrAdd(gomock.Any(), gomock.Any()).Times(0)
		mh.EXPECT().AddrDel(gomock.Any(), gomock.Any()).Times(0)
		mh.EXPECT().LinkByName(gomock.Any()).Times(0)

		ctrl.Finish()
	})

	JustBeforeEach(func() {
		manager = NewNetifManager(addr, interfaceName, manageInterface)
		dm = manager.(*netifManagerDefault)
		// override the default handler
		dm.Handle = mh
		Expect(dm).NotTo(BeNil())
	})

	Describe("NewNetifManager", func() {
		It("should return a valid Manager", func() {
			Expect(manager).NotTo(BeNil())
		})

		Context("address", func() {
			JustBeforeEach(func() {
				Expect(dm.addr).NotTo(BeNil(), "addr should always be set")
			})

			It("should set point to the corrext IP", func() {
				Expect(dm.addr.IPNet).To(Equal(&net.IPNet{IP: net.ParseIP(ip), Mask: net.CIDRMask(32, 32)}))
			})

			It("should have the correct scope", func() {
				Expect(dm.addr.Scope).To(Equal(0xfe))
			})
		})

		It("should set the correct device name", func() {
			Expect(dm.devName).To(Equal(interfaceName))
		})
	})

	Describe("RemoveIPAddress", func() {

		It("should return error when getting link", func() {
			mh.EXPECT().
				LinkByName(gomock.Eq("foo")).
				Return(nil, fmt.Errorf("err")).
				Times(1)

			err := manager.RemoveIPAddress()
			Expect(err).To(HaveOccurred())
		})

		Context("LinkByName succeeds", func() {

			BeforeEach(func() {
				mh.EXPECT().
					LinkByName(gomock.Eq("foo")).
					Return(dummy, nil).
					Times(1)
			})

			It("should return error when deleting ip address", func() {
				mh.EXPECT().
					AddrDel(gomock.Eq(dummy), gomock.Eq(addr)).
					Return(fmt.Errorf("err")).
					Times(1)

				err := manager.RemoveIPAddress()
				Expect(err).To(HaveOccurred())
			})

			It("should return already removed error", func() {
				mh.EXPECT().
					AddrDel(gomock.Eq(dummy), gomock.Eq(addr)).
					// Return(syscall.EEXIST).
					Return(syscall.ENOENT).
					Times(1)

				err := manager.RemoveIPAddress()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return no error when deleting link", func() {
				mh.EXPECT().
					AddrDel(gomock.Eq(dummy), gomock.Eq(addr)).
					Return(nil).
					Times(1)

				err := manager.RemoveIPAddress()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("EnsureIPAddress", func() {

		It("should return error when getting link", func() {
			mh.EXPECT().
				LinkByName(gomock.Eq("foo")).
				Return(nil, fmt.Errorf("err")).
				Times(1)

			err := manager.EnsureIPAddress()
			Expect(err).To(HaveOccurred())
		})

		Context("LinkByName errors but manageInterface is true", func() {
			BeforeEach(func() {
				mh.EXPECT().
					LinkByName(gomock.Eq("foo")).
					Return(nil, netlink.LinkNotFoundError{}).
					Times(1)

				mh.EXPECT().
					LinkAdd(gomock.Any()).
					Return(nil).
					Times(1)

				manageInterface = true
			})

			It("should return error when adding ip address", func() {
				mh.EXPECT().
					AddrAdd(gomock.Not(gomock.Eq(dummy)), gomock.Eq(addr)).
					Return(fmt.Errorf("err")).
					Times(1)

				err := manager.EnsureIPAddress()
				Expect(err).To(HaveOccurred())
			})

			It("should return already exists error", func() {
				mh.EXPECT().
					AddrAdd(gomock.Not(gomock.Eq(dummy)), gomock.Eq(addr)).
					Return(syscall.EEXIST).
					Times(1)

				err := manager.EnsureIPAddress()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return no error when deleting link", func() {
				mh.EXPECT().
					AddrAdd(gomock.Not(gomock.Eq(dummy)), gomock.Eq(addr)).
					Return(nil).
					Times(1)

				mh.EXPECT().
					LinkSetUp(gomock.Not(gomock.Eq(dummy))).
					Return(nil).
					Times(1)

				err := manager.EnsureIPAddress()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return error when set up of link fails", func() {
				mh.EXPECT().
					AddrAdd(gomock.Not(gomock.Eq(dummy)), gomock.Eq(addr)).
					Return(nil).
					Times(1)

				mh.EXPECT().
					LinkSetUp(gomock.Not(gomock.Eq(dummy))).
					Return(fmt.Errorf("err")).
					Times(1)

				err := manager.EnsureIPAddress()
				Expect(err).To(HaveOccurred())
			})

		})

		Context("LinkByName succeeds", func() {
			BeforeEach(func() {
				mh.EXPECT().
					LinkByName(gomock.Eq("foo")).
					Return(dummy, nil).
					Times(1)
			})

			It("should return error when adding ip address", func() {
				mh.EXPECT().
					AddrAdd(gomock.Eq(dummy), gomock.Eq(addr)).
					Return(fmt.Errorf("err")).
					Times(1)

				err := manager.EnsureIPAddress()
				Expect(err).To(HaveOccurred())
			})

			It("should return already exists error", func() {
				mh.EXPECT().
					AddrAdd(gomock.Eq(dummy), gomock.Eq(addr)).
					Return(syscall.EEXIST).
					Times(1)

				err := manager.EnsureIPAddress()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return no error when deleting link", func() {
				mh.EXPECT().
					AddrAdd(gomock.Eq(dummy), gomock.Eq(addr)).
					Return(nil).
					Times(1)

				err := manager.EnsureIPAddress()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

})
