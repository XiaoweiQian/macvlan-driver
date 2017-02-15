package drivers

import (
	"testing"

	"github.com/docker/docker/pkg/stringid"
	pluginNet "github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/netlabel"
	"github.com/docker/libnetwork/ns"
	"github.com/stretchr/testify/assert"
)

func initNetworkData() (*MacStore, *Driver, *pluginNet.CreateNetworkRequest, *network) {
	ms := &MacStore{}
	d := &Driver{
		networks: networkTable{},
		store:    ms,
	}
	r := &pluginNet.CreateNetworkRequest{
		NetworkID: "1",
		Options: map[string]interface{}{
			netlabel.GenericData: map[string]string{
				"parent": "eth0",
			},
		},
		IPv4Data: []*pluginNet.IPAMData{
			&pluginNet.IPAMData{
				Pool:    "192.168.1.0/24",
				Gateway: "192.168.1.1",
			},
		},
		IPv6Data: []*pluginNet.IPAMData{
			&pluginNet.IPAMData{
				Pool:    "fe80:0:0:0:0:0:c0a8:200/120",
				Gateway: "fe80:0:0:0:0:0:c0a8:201",
			},
		},
	}
	config := &configuration{
		ID:          "1",
		Parent:      "eth0",
		MacvlanMode: "bridge",
		Ipv4Subnets: []*ipv4Subnet{
			&ipv4Subnet{
				SubnetIP: "192.168.1.0/24",
				GwIP:     "192.168.1.1",
			},
		},
		Ipv6Subnets: []*ipv6Subnet{
			&ipv6Subnet{
				SubnetIP: "fe80:0:0:0:0:0:c0a8:200/120",
				GwIP:     "fe80:0:0:0:0:0:c0a8:201",
			},
		},
	}
	n := &network{
		id:        "1",
		driver:    d,
		endpoints: endpointTable{},
		config:    config,
	}

	return ms, d, r, n
}

func TestCreateNetworkWithOK(t *testing.T) {
	_, d, r, n := initNetworkData()
	err := d.CreateNetwork(r)
	assert.Nil(t, err)
	assert.NotEmpty(t, d.networks[r.NetworkID])
	assert.EqualValues(t, n, d.networks[r.NetworkID])
}

func TestCreateNetworkWithVlan(t *testing.T) {
	_, d, r, n := initNetworkData()
	opts := r.Options[netlabel.GenericData].(map[string]string)
	opts["parent"] = "eth0.10"
	n.config.CreatedSlaveLink = true
	n.config.Parent = "eth0.10"
	err := d.CreateNetwork(r)
	assert.Nil(t, err)
	assert.NotEmpty(t, d.networks[r.NetworkID])
	assert.EqualValues(t, n, d.networks[r.NetworkID])
	if link, err := ns.NlHandle().LinkByName(n.config.Parent); err == nil {
		ns.NlHandle().LinkDel(link)
	}

}

func TestCreateNetworkWithInternal(t *testing.T) {
	_, d, r, n := initNetworkData()
	r.Options[netlabel.Internal] = map[string]string{
		"internal": "true",
	}
	n.config.CreatedSlaveLink = true
	n.config.Internal = true
	n.config.Parent = "dm-1"
	err := d.CreateNetwork(r)
	assert.Nil(t, err)
	assert.NotEmpty(t, d.networks[r.NetworkID])
	assert.EqualValues(t, n, d.networks[r.NetworkID])
	if link, err := ns.NlHandle().LinkByName(n.config.Parent); err == nil {
		ns.NlHandle().LinkDel(link)
	}

}

func TestDeleteNetworkWithVlan(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	c := d.networks[r.NetworkID].config
	c.CreatedSlaveLink = true
	c.Parent = "eth0.10"
	err := createVlanLink(c.Parent)
	assert.Nil(t, err)
	ms.On("StoreDelete", ep).Return(nil)
	dr := &pluginNet.DeleteNetworkRequest{
		NetworkID: r.NetworkID,
	}
	err = d.DeleteNetwork(dr)
	assert.Nil(t, err)
	assert.Empty(t, d.networks[r.NetworkID])
}

func TestDeleteNetworkWithInternal(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	c := d.networks[r.NetworkID].config
	c.CreatedSlaveLink = true
	c.Internal = true
	c.Parent = "dm-1"
	err := createDummyLink(c.Parent, getDummyName(stringid.TruncateID(c.ID)))
	assert.Nil(t, err)
	ms.On("StoreDelete", ep).Return(nil)
	dr := &pluginNet.DeleteNetworkRequest{
		NetworkID: r.NetworkID,
	}
	err = d.DeleteNetwork(dr)
	assert.Nil(t, err)
	assert.Empty(t, d.networks[r.NetworkID])
}
