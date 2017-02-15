package drivers

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/Sirupsen/logrus"
	pluginNet "github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/datastore"
	"github.com/docker/libnetwork/netlabel"
	"github.com/docker/libnetwork/netutils"
	"github.com/docker/libnetwork/ns"
	"github.com/docker/libnetwork/osl"
	"github.com/docker/libnetwork/types"
)

const (
	macvlanPrefix         = "macvlan"
	macvlanEndpointPrefix = macvlanPrefix + "/endpoint"
)

type endpoint struct {
	id       string
	nid      string
	mac      net.HardwareAddr
	addr     *net.IPNet
	addrv6   *net.IPNet
	srcName  string
	dbIndex  uint64
	dbExists bool
}

// CreateEndpoint assigns the mac, ip and endpoint id for the new container
func (d *Driver) CreateEndpoint(r *pluginNet.CreateEndpointRequest) (*pluginNet.CreateEndpointResponse, error) {
	logrus.Debugf("CreateEndpoint macvlan : interface info %s", r.Interface)
	defer osl.InitOSContext()()
	networkID := r.NetworkID
	if networkID == "" {
		return nil, fmt.Errorf("invalid network id passed while create macvlan endpoint")
	}
	endpointID := r.EndpointID
	if endpointID == "" {
		return nil, fmt.Errorf("invalid endpoint id passed while create macvlan endpoint")
	}
	intf := r.Interface
	if intf == nil {
		return nil, fmt.Errorf("invalid interface passed while create macvlan endpoint")
	}
	n, ok := d.networks[networkID]
	if !ok {
		return nil, fmt.Errorf("macvlan network with id %s not found", networkID)
	}
	var addrNet, addrv6Net *net.IPNet
	addr, mask, _ := net.ParseCIDR(intf.Address)
	if addr != nil && mask != nil {
		addrNet = &net.IPNet{IP: addr, Mask: mask.Mask}
	}
	addrv6, maskv6, _ := net.ParseCIDR(intf.AddressIPv6)
	if addrv6 != nil && maskv6 != nil {
		addrv6Net = &net.IPNet{IP: addrv6, Mask: maskv6.Mask}
	}
	mac, _ := net.ParseMAC(intf.MacAddress)
	ep := &endpoint{
		id:     endpointID,
		nid:    networkID,
		addr:   addrNet,
		addrv6: addrv6Net,
		mac:    mac,
	}
	if ep.addr == nil {
		return nil, fmt.Errorf("create endpoint was not passed interface IP address")
	}

	if ep.mac == nil {
		ep.mac = netutils.GenerateMACFromIP(ep.addr.IP)
		intf.MacAddress = ep.mac.String()
		logrus.Debugf("CreateEndpoint: generate mac ip=%s,mac=%s,eth=%s", ep.addr.IP.String(), ep.mac.String())
	}

	epOptions := r.Options
	// disallow portmapping -p
	if opt, ok := epOptions[netlabel.PortMap]; ok {
		if _, ok := opt.([]types.PortBinding); ok {
			if len(opt.([]types.PortBinding)) > 0 {
				logrus.Warnf("%s driver does not support port mappings", macvlanType)
			}
		}
	}

	// disallow port exposure --expose
	if opt, ok := epOptions[netlabel.ExposedPorts]; ok {
		if _, ok := opt.([]types.TransportPort); ok {
			if len(opt.([]types.TransportPort)) > 0 {
				logrus.Warnf("%s driver does not support port exposures", macvlanType)
			}
		}
	}

	if err := d.store.StoreUpdate(ep); err != nil {
		return nil, fmt.Errorf("failed to save macvlan endpoint %s to store: %v", ep.id[0:7], err)
	}

	n.addEndpoint(ep)

	epResponse := &pluginNet.CreateEndpointResponse{Interface: &pluginNet.EndpointInterface{"", "", intf.MacAddress}}
	return epResponse, nil
}

// DeleteEndpoint removes the endpoint and associated netlink interface
func (d *Driver) DeleteEndpoint(r *pluginNet.DeleteEndpointRequest) error {
	logrus.Debugf("DeleteEndpoint macvlan")
	defer osl.InitOSContext()()
	nid := r.NetworkID
	eid := r.EndpointID
	if nid == "" {
		return fmt.Errorf("invalid network id")
	}
	if eid == "" {
		return fmt.Errorf("invalid endpoint id")
	}
	n := d.networks[nid]
	if n == nil {
		return fmt.Errorf("network id %q not found", nid)
	}
	ep := n.endpoints[eid]
	if ep == nil {
		return fmt.Errorf("endpoint id %q not found", eid)
	}
	if err := d.deleteEndpoint(n, ep); err != nil {
		return err
	}

	return nil
}

func (d *Driver) deleteEndpoint(n *network, ep *endpoint) error {
	if link, err := ns.NlHandle().LinkByName(ep.srcName); err == nil {
		ns.NlHandle().LinkDel(link)
	}
	if err := d.store.StoreDelete(ep); err != nil {
		return fmt.Errorf("failed to remove macvlan endpoint %s to store: %v", ep.id[0:7], err)
	}
	n.deleteEndpoint(ep.id)

	return nil
}

// EndpointInfo ...
func (d *Driver) EndpointInfo(r *pluginNet.InfoRequest) (*pluginNet.InfoResponse, error) {
	logrus.Debugf("EndpointInfo macvlan")
	res := &pluginNet.InfoResponse{
		Value: make(map[string]string),
	}
	return res, nil
}

func (ep *endpoint) MarshalJSON() ([]byte, error) {
	epMap := make(map[string]interface{})
	epMap["id"] = ep.id
	epMap["nid"] = ep.nid
	epMap["SrcName"] = ep.srcName
	if len(ep.mac) != 0 {
		epMap["MacAddress"] = ep.mac.String()
	}
	if ep.addr != nil {
		epMap["Addr"] = ep.addr.String()
	}
	if ep.addrv6 != nil {
		epMap["Addrv6"] = ep.addrv6.String()
	}
	return json.Marshal(epMap)
}

func (ep *endpoint) UnmarshalJSON(b []byte) error {
	var (
		err   error
		epMap map[string]interface{}
	)

	if err = json.Unmarshal(b, &epMap); err != nil {
		return fmt.Errorf("Failed to unmarshal to macvlan endpoint: %v", err)
	}

	if v, ok := epMap["MacAddress"]; ok {
		if ep.mac, err = net.ParseMAC(v.(string)); err != nil {
			return types.InternalErrorf("failed to decode macvlan endpoint MAC address (%s) after json unmarshal: %v", v.(string), err)
		}
	}
	if v, ok := epMap["Addr"]; ok {
		if ep.addr, err = types.ParseCIDR(v.(string)); err != nil {
			return types.InternalErrorf("failed to decode macvlan endpoint IPv4 address (%s) after json unmarshal: %v", v.(string), err)
		}
	}
	if v, ok := epMap["Addrv6"]; ok {
		if ep.addrv6, err = types.ParseCIDR(v.(string)); err != nil {
			return types.InternalErrorf("failed to decode macvlan endpoint IPv6 address (%s) after json unmarshal: %v", v.(string), err)
		}
	}
	ep.id = epMap["id"].(string)
	ep.nid = epMap["nid"].(string)
	ep.srcName = epMap["SrcName"].(string)

	return nil
}

func (ep *endpoint) Key() []string {
	return []string{macvlanEndpointPrefix, ep.id}
}

func (ep *endpoint) KeyPrefix() []string {
	return []string{macvlanEndpointPrefix}
}

func (ep *endpoint) Value() []byte {
	b, err := json.Marshal(ep)
	if err != nil {
		return nil
	}
	return b
}

func (ep *endpoint) SetValue(value []byte) error {
	return json.Unmarshal(value, ep)
}

func (ep *endpoint) Index() uint64 {
	return ep.dbIndex
}

func (ep *endpoint) SetIndex(index uint64) {
	ep.dbIndex = index
	ep.dbExists = true
}

func (ep *endpoint) Exists() bool {
	return ep.dbExists
}

func (ep *endpoint) Skip() bool {
	return false
}

func (ep *endpoint) New() datastore.KVObject {
	return &endpoint{}
}

func (ep *endpoint) CopyTo(o datastore.KVObject) error {
	dstEp := o.(*endpoint)
	*dstEp = *ep
	return nil
}

func (ep *endpoint) DataScope() string {
	return datastore.LocalScope
}
