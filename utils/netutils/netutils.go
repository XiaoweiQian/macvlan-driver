package netutils

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/types"
	"github.com/vishvananda/netlink"
)

func genMAC(ip net.IP) net.HardwareAddr {
	hw := make(net.HardwareAddr, 6)
	// The first byte of the MAC address has to comply with these rules:
	// 1. Unicast: Set the least-significant bit to 0.
	// 2. Address is locally administered: Set the second-least-significant bit (U/L) to 1.
	hw[0] = 0x02
	// The first 24 bits of the MAC represent the Organizationally Unique Identifier (OUI).
	// Since this address is locally administered, we can do whatever we want as long as
	// it doesn't conflict with other addresses.
	hw[1] = 0x42
	// Fill the remaining 4 bytes based on the input
	if ip == nil {
		rand.Read(hw[2:])
	} else {
		copy(hw[2:], ip.To4())
	}
	return hw
}

// GenerateRandomMAC returns a new 6-byte(48-bit) hardware address (MAC)
func GenerateRandomMAC() net.HardwareAddr {
	return genMAC(nil)
}

// GenerateMACFromIP returns a locally administered MAC address where the 4 least
// significant bytes are derived from the IPv4 address.
func GenerateMACFromIP(ip net.IP) net.HardwareAddr {
	return genMAC(ip)
}

// GenerateRandomName returns a new name joined with a prefix.  This size
// specified is used to truncate the randomly generated value
func GenerateRandomName(prefix string, size int) (string, error) {
	id := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(id)[:size], nil
}

// GenerateIfaceName returns an interface name using the passed in
// prefix and the length of random bytes. The api ensures that the
// there are is no interface which exists with that name.
func GenerateIfaceName(nlh *netlink.Handle, prefix string, len int) (string, error) {
	linkByName := netlink.LinkByName
	if nlh != nil {
		linkByName = nlh.LinkByName
	}
	for i := 0; i < 3; i++ {
		name, err := GenerateRandomName(prefix, len)
		if err != nil {
			continue
		}
		_, err = linkByName(name)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return name, nil
			}
			return "", err
		}
	}
	return "", types.InternalErrorf("could not generate interface name")
}

// SetInterfaceIP  Set IP address of an interface
func SetInterfaceIP(name string, ipstr string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	ipaddr, err := netlink.ParseAddr(ipstr)
	if err != nil {
		return err
	}
	netlink.LinkSetUp(iface)
	return netlink.AddrAdd(iface, ipaddr)
}

// SetInterfaceMac  Set mac address of an interface
func SetInterfaceMac(name string, macaddr string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	hwaddr, err := net.ParseMAC(macaddr)
	if err != nil {
		return err
	}
	return netlink.LinkSetHardwareAddr(iface, hwaddr)
}

// CreateVethPair creates veth interface pairs with specified name
func CreateVethPair(name1, name2 string) error {
	logrus.Infof("Creating Veth pairs with name: %s, %s", name1, name2)

	// Veth pair params
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:   name1,
			TxQLen: 0,
		},
		PeerName: name2,
	}

	// Create the veth pair
	if err := netlink.LinkAdd(veth); err != nil {
		logrus.Errorf("error creating veth pair: %v", err)
		return err
	}

	return nil
}

// DeleteVethPair deletes veth interface pairs
func DeleteVethPair(name1, name2 string) error {
	logrus.Infof("Deleting Veth pairs with name: %s, %s", name1, name2)

	// Veth pair params
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:   name1,
			TxQLen: 0,
		},
		PeerName: name2,
	}

	// Create the veth pair
	if err := netlink.LinkDel(veth); err != nil {
		return err
	}

	return nil
}

// SetLinkUp sets the link up
func SetLinkUp(name string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetUp(iface)
}
