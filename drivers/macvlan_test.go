package drivers

import (
	"fmt"
	"testing"

	pluginNet "github.com/docker/go-plugins-helpers/network"
	"github.com/stretchr/testify/assert"
)

func initData() (*MacStore, *Driver, *pluginNet.AllocateNetworkRequest, *network) {
	ms := &MacStore{}
	d := &Driver{
		networks: networkTable{},
		store:    ms,
	}
	r := &pluginNet.AllocateNetworkRequest{
		NetworkID: "1",
		Options: map[string]string{
			"parent": "eth0",
		},
		IPv4Data: []pluginNet.IPAMData{
			pluginNet.IPAMData{
				Pool:    "192.168.1.0/24",
				Gateway: "192.168.1.1",
			},
		},
		IPv6Data: []pluginNet.IPAMData{
			pluginNet.IPAMData{
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
		id:     "1",
		driver: d,
		config: config,
	}

	return ms, d, r, n
}

func TestInitWithOK(t *testing.T) {
	ms, d, _, _ := initData()
	ms.On("InitStore", d).Return(nil)
	d, err := Init(d)
	assert.Nil(t, err)
	assert.NotNil(t, d)
}

func TestInitWithErr(t *testing.T) {
	ms, d, _, _ := initData()
	ms.On("InitStore", d).Return(fmt.Errorf("error"))
	d, err := Init(d)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Failure during init macvlan local store: error")
	assert.Nil(t, d)
}

func TestGetCapabilities(t *testing.T) {
	_, d, _, _ := initData()
	cap, err := d.GetCapabilities()
	assert.Nil(t, err)
	assert.EqualValues(t, pluginNet.GlobalScope, cap.Scope)
}

func TestAllocateNetworkWithOK(t *testing.T) {
	_, d, r, n := initData()
	res, err := d.AllocateNetwork(r)
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.EqualValues(t, n, d.networks[r.NetworkID])
}

func TestAllocateNetworkWithMode(t *testing.T) {
	_, d, r, n := initData()
	r.Options["macvlan_mode"] = "private"
	n.config.MacvlanMode = "private"
	res, err := d.AllocateNetwork(r)
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.EqualValues(t, n, d.networks[r.NetworkID])
}

func TestAllocateNetworkWithInvalidSubnet(t *testing.T) {
	_, d, r, _ := initData()
	r.IPv4Data[0].Pool = "0.0.0.0/0"
	res, err := d.AllocateNetwork(r)
	assert.NotNil(t, err)
	assert.Nil(t, res)
	assert.EqualError(t, err, "ipv4 pool is empty")
}

func TestAllocateNetworkWithInvalidID(t *testing.T) {
	_, d, r, _ := initData()
	r.NetworkID = ""
	res, err := d.AllocateNetwork(r)
	assert.NotNil(t, err)
	assert.Nil(t, res)
	assert.EqualError(t, err, "invalid network id for macvlan network")
}

func TestAllocateNetworkWithSameParent(t *testing.T) {
	_, d, r, n := initData()
	d.networks[r.NetworkID] = n
	res, err := d.AllocateNetwork(r)
	assert.NotNil(t, err)
	assert.Nil(t, res)
	assert.EqualError(t, err, "network dm-1 is already using parent interface eth0")
}

func TestAllocateNetworkWithInvalidParent(t *testing.T) {
	_, d, r, _ := initData()
	r.Options["parent"] = "lo"
	res, err := d.AllocateNetwork(r)
	assert.NotNil(t, err)
	assert.Nil(t, res)
	assert.EqualError(t, err, "loopback interface is not a valid macvlan parent link")
}

func TestFreeNetworkWithOK(t *testing.T) {
	_, d, r, n := initData()
	d.networks[r.NetworkID] = n
	fnr := &pluginNet.FreeNetworkRequest{
		NetworkID: r.NetworkID,
	}
	err := d.FreeNetwork(fnr)
	assert.Empty(t, d.networks[r.NetworkID])
	assert.Nil(t, err)
}

func TestFreeNetworkWithNotFound(t *testing.T) {
	_, d, r, n := initData()
	d.networks[r.NetworkID] = n
	fnr := &pluginNet.FreeNetworkRequest{
		NetworkID: "2",
	}
	err := d.FreeNetwork(fnr)
	assert.NotEmpty(t, d.networks[r.NetworkID])
	assert.Nil(t, err)
}
