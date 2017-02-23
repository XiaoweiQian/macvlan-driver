package drivers

import (
	"fmt"
	"net"

	"github.com/Sirupsen/logrus"
	"github.com/XiaoweiQian/macvlan-driver/utils/netutils"
	pluginNet "github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/ns"
	"github.com/docker/libnetwork/osl"
)

// Join method is invoked when a Sandbox is attached to an endpoint.
func (d *Driver) Join(r *pluginNet.JoinRequest) (*pluginNet.JoinResponse, error) {
	defer osl.InitOSContext()()
	nid := r.NetworkID
	eid := r.EndpointID
	logrus.Infof("Join macvlan nid=%s,eid=%s", nid, eid)
	n, err := d.getNetwork(nid)
	if err != nil {
		return nil, err
	}
	ep := n.endpoint(eid)
	if ep == nil {
		str := fmt.Sprintf("could not find endpoint with id %s", eid)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}
	// generate a name for the iface that will be renamed to eth0 in the sbox
	containerIfName, err := netutils.GenerateIfaceName(ns.NlHandle(), vethPrefix, vethLen)
	if err != nil {
		str := fmt.Sprintf("error generating an interface name: %s", err)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}
	// create the netlink macvlan interface
	vethName, err := createMacVlan(containerIfName, n.config.Parent, n.config.MacvlanMode)
	if err != nil {
		str := fmt.Sprintf("Join: createMacVlan error: %s", err)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}

	// parse and match the endpoint address with the available v4 subnets
	var v4gwStr, v6gwStr string
	if len(n.config.Ipv4Subnets) > 0 {
		s := n.getSubnetforIPv4(ep.addr)
		if s == nil {
			str := fmt.Sprintf("could not find a valid ipv4 subnet for endpoint %s", eid)
			logrus.Errorf(str)
			return nil, fmt.Errorf(str)
		}
		v4gw, _, err := net.ParseCIDR(s.GwIP)
		if err != nil {
			str := fmt.Sprintf("gatway %s is not a valid ipv4 address: %v", s.GwIP, err)
			logrus.Errorf(str)
			return nil, fmt.Errorf(str)
		}
		v4gwStr = v4gw.String()
		logrus.Infof("Macvlan Endpoint Joined with IPv4_Addr: %s, Gateway: %s, MacVlan_Mode: %s, Parent: %s",
			ep.addr.IP.String(), v4gw.String(), n.config.MacvlanMode, n.config.Parent)
	}
	// parse and match the endpoint address with the available v6 subnets
	if len(n.config.Ipv6Subnets) > 0 {
		s := n.getSubnetforIPv6(ep.addrv6)
		if s == nil {
			return nil, fmt.Errorf("could not find a valid ipv6 subnet for endpoint %s", eid)
		}
		v6gw, _, err := net.ParseCIDR(s.GwIP)
		if err != nil {
			return nil, fmt.Errorf("gatway %s is not a valid ipv6 address: %v", s.GwIP, err)
		}
		v6gwStr = v6gw.String()
		logrus.Infof("Macvlan Endpoint Joined with IPv6_Addr: %s Gateway: %s MacVlan_Mode: %s, Parent: %s",
			ep.addrv6.IP.String(), v6gw.String(), n.config.MacvlanMode, n.config.Parent)
	}
	if err := d.store.StoreUpdate(ep); err != nil {
		str := fmt.Sprintf("failed to save macvlan endpoint %s to store: %v", ep.id[0:7], err)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}
	// bind the generated iface name to the endpoint
	ep.srcName = vethName

	res := &pluginNet.JoinResponse{
		InterfaceName: pluginNet.InterfaceName{
			SrcName:   ep.srcName,
			DstPrefix: containerVethPrefix,
		},
		Gateway:     v4gwStr,
		GatewayIPv6: v6gwStr,
	}
	return res, nil
}

// Leave method is invoked when a Sandbox detaches from an endpoint.
func (d *Driver) Leave(r *pluginNet.LeaveRequest) error {
	nid := r.NetworkID
	eid := r.EndpointID
	logrus.Infof("Leave macvlan nid=%s,eid=%s", nid, eid)
	if nid == "" {
		return fmt.Errorf("invalid network id")
	}
	if eid == "" {
		return fmt.Errorf("invalid endpoint id")
	}
	n, err := d.getNetwork(nid)
	if err != nil {
		return err
	}
	ep, err := n.getEndpoint(eid)
	if err != nil {
		return err
	}
	if ep == nil {
		return fmt.Errorf("could not find endpoint with id %s", eid)
	}

	return nil
}

// getSubnetforIP returns the ipv4 subnet to which the given IP belongs
func (n *network) getSubnetforIPv4(ip *net.IPNet) *ipv4Subnet {
	for _, s := range n.config.Ipv4Subnets {
		_, snet, err := net.ParseCIDR(s.SubnetIP)
		if err != nil {
			return nil
		}
		// first check if the mask lengths are the same
		i, _ := snet.Mask.Size()
		j, _ := ip.Mask.Size()
		if i != j {
			continue
		}
		if snet.Contains(ip.IP) {
			return s
		}
	}

	return nil
}

// getSubnetforIPv6 returns the ipv6 subnet to which the given IP belongs
func (n *network) getSubnetforIPv6(ip *net.IPNet) *ipv6Subnet {
	for _, s := range n.config.Ipv6Subnets {
		_, snet, err := net.ParseCIDR(s.SubnetIP)
		if err != nil {
			return nil
		}
		// first check if the mask lengths are the same
		i, _ := snet.Mask.Size()
		j, _ := ip.Mask.Size()
		if i != j {
			continue
		}
		if snet.Contains(ip.IP) {
			return s
		}
	}

	return nil
}
