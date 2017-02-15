package drivers

import (
	"fmt"
	"net"
	"testing"

	pluginNet "github.com/docker/go-plugins-helpers/network"
	"github.com/stretchr/testify/assert"
)

func initEndpointData() (*MacStore, *Driver, *pluginNet.CreateEndpointRequest, *endpoint) {
	ms := &MacStore{}
	d := &Driver{
		networks: networkTable{},
		store:    ms,
	}
	config := &configuration{
		ID:          "1",
		Parent:      "eth0",
		MacvlanMode: "bridge",
		Ipv4Subnets: []*ipv4Subnet{
			&ipv4Subnet{
				SubnetIP: "192.168.2.0/24",
				GwIP:     "192.168.2.1/24",
			},
		},
		Ipv6Subnets: []*ipv6Subnet{
			&ipv6Subnet{
				SubnetIP: "fe80:0:0:0:0:0:c0a8:200/120",
				GwIP:     "fe80:0:0:0:0:0:c0a8:201/120",
			},
		},
	}
	n := &network{
		id:        "1",
		driver:    d,
		config:    config,
		endpoints: endpointTable{},
	}
	d.networks[n.id] = n

	r := &pluginNet.CreateEndpointRequest{
		NetworkID:  "1",
		EndpointID: "1234567",
		Interface: &pluginNet.EndpointInterface{
			Address:     "192.168.2.2/24",
			AddressIPv6: "fe80:0:0:0:0:0:c0a8:202/120",
		},
		Options: map[string]interface{}{
			"parent": "eth0",
		},
	}
	addr, mask, _ := net.ParseCIDR(r.Interface.Address)
	addrNet := &net.IPNet{IP: addr, Mask: mask.Mask}
	addrv6, maskv6, _ := net.ParseCIDR(r.Interface.AddressIPv6)
	addrv6Net := &net.IPNet{IP: addrv6, Mask: maskv6.Mask}
	mac, _ := net.ParseMAC("02:42:c0:a8:02:02")
	ep := &endpoint{
		id:     r.EndpointID,
		nid:    r.NetworkID,
		addr:   addrNet,
		addrv6: addrv6Net,
		mac:    mac,
	}
	return ms, d, r, ep
}

func TestCreateEndpointWithOK(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	ms.On("StoreUpdate", ep).Return(nil)
	res, err := d.CreateEndpoint(r)
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.EqualValues(t, ep, d.networks[ep.nid].endpoints[ep.id])
}

func TestCreateEndpointWithErr(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	ms.On("StoreUpdate", ep).Return(fmt.Errorf("error"))
	res, err := d.CreateEndpoint(r)
	assert.NotNil(t, err)
	assert.Nil(t, res)
	assert.EqualError(t, err, "failed to save macvlan endpoint 1234567 to store: error")
}

func TestDeleteEndpointWithOK(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	ms.On("StoreDelete", ep).Return(nil)
	der := &pluginNet.DeleteEndpointRequest{
		NetworkID:  r.NetworkID,
		EndpointID: r.EndpointID,
	}
	err := d.DeleteEndpoint(der)
	assert.Nil(t, err)
	assert.Empty(t, d.networks[r.NetworkID].endpoints[r.EndpointID])
}

func TestDeleteEndpointWithErr(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	ms.On("StoreDelete", ep).Return(fmt.Errorf("error"))
	der := &pluginNet.DeleteEndpointRequest{
		NetworkID:  r.NetworkID,
		EndpointID: r.EndpointID,
	}
	err := d.DeleteEndpoint(der)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "failed to remove macvlan endpoint 1234567 to store: error")
	assert.NotEmpty(t, d.networks[r.NetworkID].endpoints[r.EndpointID])
}

func TestEndpointInfo(t *testing.T) {
	_, d, r, _ := initEndpointData()
	ir := &pluginNet.InfoRequest{
		NetworkID:  r.NetworkID,
		EndpointID: r.EndpointID,
	}
	res, err := d.EndpointInfo(ir)
	assert.Nil(t, err)
	assert.NotNil(t, res)
}

func TestMarshaJSON(t *testing.T) {
	_, _, _, ep := initEndpointData()
	b, err := ep.MarshalJSON()
	assert.Nil(t, err)
	ep1 := &endpoint{}
	err1 := ep1.UnmarshalJSON(b)
	assert.Nil(t, err1)
	assert.EqualValues(t, ep1, ep)
}
