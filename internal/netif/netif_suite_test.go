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
	"fmt"
	"net"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func TestNetif(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Netif Suite")
}

var _ = Describe("Manager", func() {

	var (
		ctrl          *gomock.Controller
		mh            *MockHandle
		ip            net.IP
		interfaceName string
		manager       Manager
		dm            *netifManagerDefault
		dummy         *netlink.Dummy
	)

	BeforeEach(func() {
		ip = net.ParseIP("192.168.0.3")
		interfaceName = "foo"
		ctrl = gomock.NewController(GinkgoT())
		mh = NewMockHandle(ctrl)
		dummy = &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{Name: interfaceName},
		}

	})

	AfterEach(func() {

		mh.EXPECT().AddrAdd(gomock.Any(), gomock.Any()).Times(0)
		mh.EXPECT().AddrDel(gomock.Any(), gomock.Any()).Times(0)
		mh.EXPECT().AddrList(gomock.Any(), gomock.Any()).Times(0)
		mh.EXPECT().LinkAdd(gomock.Any()).Times(0)
		mh.EXPECT().LinkByName(gomock.Any()).Times(0)
		mh.EXPECT().LinkDel(gomock.Any()).Times(0)

		ctrl.Finish()
	})

	JustBeforeEach(func() {
		manager = NewNetifManager(ip, interfaceName)
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
				Expect(dm.addr.IPNet).To(Equal(&net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}))
			})

			It("should have the correct scope", func() {
				Expect(dm.addr.Scope).To(Equal(0xfe))
			})
		})

		It("should set the correct device name", func() {
			Expect(dm.devName).To(Equal(interfaceName))
		})
	})

	Describe("RemoveDummyDevice", func() {

		It("should return error when getting link", func() {
			mh.EXPECT().
				LinkByName(gomock.Eq("foo")).
				Return(nil, fmt.Errorf("err")).
				Times(1)

			err := manager.RemoveDummyDevice()
			Expect(err).To(HaveOccurred())
		})

		Context("LinkByName succeeds", func() {

			BeforeEach(func() {
				mh.EXPECT().
					LinkByName(gomock.Eq("foo")).
					Return(dummy, nil).
					Times(1)
			})

			It("should return error when deleting link", func() {
				mh.EXPECT().
					LinkDel(gomock.Eq(dummy)).
					Return(fmt.Errorf("err")).
					Times(1)

				err := manager.RemoveDummyDevice()
				Expect(err).To(HaveOccurred())
			})

			It("should return no error when deleting link", func() {
				mh.EXPECT().
					LinkDel(gomock.Eq(dummy)).
					Return(nil).
					Times(1)

				err := manager.RemoveDummyDevice()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("EnsureDummyDevice", func() {

		Context("error when getting link", func() {

			Context("random error", func() {

				It("should exit without any other operations", func() {
					mh.EXPECT().
						LinkByName(gomock.Eq("foo")).
						Return(nil, fmt.Errorf("err")).
						Times(1)

					Expect(manager.EnsureDummyDevice()).To(HaveOccurred())
				})
			})

			Context("LinkNotFoundError", func() {

				BeforeEach(func() {
					mh.EXPECT().
						LinkByName(gomock.Eq("foo")).
						Return(nil, netlink.LinkNotFoundError{}). // this throws nil pointer in the code
						Times(1)
				})

				It("adds link error", func() {
					mh.EXPECT().
						LinkAdd(dummy).
						Return(fmt.Errorf("error when creating link")).
						Times(1)

					Expect(manager.EnsureDummyDevice()).To(HaveOccurred())
				})

				Context("link add success", func() {

					BeforeEach(func() {
						mh.EXPECT().
							LinkAdd(gomock.Eq(dummy)).
							Return(nil).
							Times(1)
					})

					It("add address error", func() {
						mh.EXPECT().
							AddrAdd(gomock.Eq(dummy), gomock.Eq(dm.addr)).
							Return(fmt.Errorf("error when adding addr")).
							Times(1)

						Expect(manager.EnsureDummyDevice()).To(HaveOccurred())
					})

					It("add address succeeds", func() {
						mh.EXPECT().
							AddrAdd(gomock.Eq(dummy), gomock.Eq(dm.addr)).
							Return(nil).
							Times(1)

						Expect(manager.EnsureDummyDevice()).NotTo(HaveOccurred())
					})
				})

			})
		})

		Context("interface is not of type dummy", func() {
			var notDummy *netlink.Vlan
			BeforeEach(func() {
				notDummy = &netlink.Vlan{}
				mh.EXPECT().
					LinkByName(gomock.Eq("foo")).
					Return(notDummy, nil).
					Times(1)
			})

			It("delete link with error", func() {
				mh.EXPECT().
					LinkDel(gomock.Eq(notDummy)).
					Return(fmt.Errorf("some error when deleting")).
					Times(1)

				Expect(manager.EnsureDummyDevice()).To(HaveOccurred())
			})

			It("delete and recreate without error", func() {
				mh.EXPECT().
					LinkDel(gomock.Eq(notDummy)).
					Return(nil).
					Times(1)
				mh.EXPECT().
					LinkAdd(gomock.Eq(dummy)).
					Return(nil).
					Times(1)

				mh.EXPECT().
					AddrAdd(gomock.Eq(dummy), gomock.Eq(dm.addr)).
					Return(nil).
					Times(1)

				Expect(manager.EnsureDummyDevice()).ToNot(HaveOccurred())
			})
		})

		Context("existing dummy link found", func() {
			BeforeEach(func() {
				mh.EXPECT().
					LinkByName(gomock.Eq("foo")).
					Return(dummy, nil).
					Times(1)
			})

			It("returns error when listing addresses", func() {
				mh.EXPECT().
					AddrList(gomock.Eq(dummy), gomock.Eq(unix.AF_INET)).
					Return(nil, fmt.Errorf("error listing addresses")).
					Times(1)

				Expect(manager.EnsureDummyDevice()).To(HaveOccurred())
			})

			It("do nothing as address already exists", func() {
				mh.EXPECT().
					AddrList(gomock.Eq(dummy), gomock.Eq(unix.AF_INET)).
					Return([]netlink.Addr{*dm.addr}, nil).
					Times(1)

				Expect(manager.EnsureDummyDevice()).ToNot(HaveOccurred())
			})

			Context("add missing address", func() {
				BeforeEach(func() {
					mh.EXPECT().
						AddrList(gomock.Eq(dummy), gomock.Eq(unix.AF_INET)).
						Return([]netlink.Addr{}, nil).
						Times(1)

				})

				It("successfully", func() {
					mh.EXPECT().
						AddrAdd(gomock.Eq(dummy), gomock.Eq(dm.addr)).
						Return(nil).
						Times(1)

					Expect(manager.EnsureDummyDevice()).ToNot(HaveOccurred())
				})

				It("with error", func() {
					mh.EXPECT().
						AddrAdd(gomock.Eq(dummy), gomock.Eq(dm.addr)).
						Return(fmt.Errorf("error for adding address")).
						Times(1)

					Expect(manager.EnsureDummyDevice()).To(HaveOccurred())
				})
			})

			Context("add missing address", func() {
				var extraAddr netlink.Addr

				BeforeEach(func() {
					extraAddr = netlink.Addr{
						IPNet: netlink.NewIPNet(net.ParseIP("1.1.1.1")),
					}

					mh.EXPECT().
						AddrList(gomock.Eq(dummy), gomock.Eq(unix.AF_INET)).
						Return([]netlink.Addr{extraAddr}, nil).
						Times(1)
				})

				It("delete extra", func() {

					mh.EXPECT().
						AddrDel(gomock.Eq(dummy), gomock.Eq(&extraAddr)).
						Return(nil).
						Times(1)

					mh.EXPECT().
						AddrAdd(gomock.Eq(dummy), gomock.Eq(dm.addr)).
						Return(nil).
						Times(1)

					Expect(manager.EnsureDummyDevice()).ToNot(HaveOccurred())
				})

				It("delete extra with error", func() {

					mh.EXPECT().
						AddrDel(gomock.Eq(dummy), gomock.Eq(&extraAddr)).
						Return(fmt.Errorf("error deleting extra ip")).
						Times(1)

					Expect(manager.EnsureDummyDevice()).To(HaveOccurred())
				})
			})

		})
	})

})
