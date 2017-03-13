package drivers

import (
	"testing"

	pluginNet "github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/ns"
	"github.com/stretchr/testify/assert"
)

func TestJoinWithOK(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	ms.On("StoreUpdate", ep).Return(nil)
	jr := &pluginNet.JoinRequest{
		NetworkID:  r.NetworkID,
		EndpointID: r.EndpointID,
	}
	res, err := d.Join(jr)
	defer func() {
		if link, err := ns.NlHandle().LinkByName(res.InterfaceName.SrcName); err == nil {
			ns.NlHandle().LinkDel(link)
		}
	}()
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.NotEmpty(t, res.InterfaceName)
	assert.NotEmpty(t, ep.srcName)
}

func TestJoinWithInvalidNetworkId(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	ms.On("StoreUpdate", ep).Return(nil)
	jr := &pluginNet.JoinRequest{
		NetworkID:  "",
		EndpointID: r.EndpointID,
	}
	res, err := d.Join(jr)
	assert.Nil(t, res)
	assert.NotNil(t, err)

}

func TestJoinWithInvalidEndpointId(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	ms.On("StoreUpdate", ep).Return(nil)
	jr := &pluginNet.JoinRequest{
		NetworkID:  r.NetworkID,
		EndpointID: "100",
	}
	res, err := d.Join(jr)
	assert.Nil(t, res)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "could not find endpoint with id 100")

}

func TestJoinWithParentNotFound(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	d.networks[r.NetworkID].config.Parent = "eth100"
	ms.On("StoreUpdate", ep).Return(nil)
	jr := &pluginNet.JoinRequest{
		NetworkID:  r.NetworkID,
		EndpointID: r.EndpointID,
	}
	res, err := d.Join(jr)
	assert.Nil(t, res)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Join: createMacVlan error: the requested parent interface eth100 was not found on the Docker host")

}

func TestJoinWithInvalidIPV4(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	ep.addr = nil
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	ms.On("StoreUpdate", ep).Return(nil)
	jr := &pluginNet.JoinRequest{
		NetworkID:  r.NetworkID,
		EndpointID: r.EndpointID,
	}
	res, err := d.Join(jr)
	assert.Nil(t, res)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "could not find a valid ipv4 subnet for endpoint 1234567")

}

func TestJoinWithInvalidIPV6(t *testing.T) {
	ms, d, r, ep := initEndpointData()
	ep.addrv6 = nil
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	ms.On("StoreUpdate", ep).Return(nil)
	jr := &pluginNet.JoinRequest{
		NetworkID:  r.NetworkID,
		EndpointID: r.EndpointID,
	}
	res, err := d.Join(jr)
	assert.Nil(t, res)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "could not find a valid ipv6 subnet for endpoint 1234567")

}

func TestLeaveWithOK(t *testing.T) {
	_, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	lr := &pluginNet.LeaveRequest{
		NetworkID:  r.NetworkID,
		EndpointID: r.EndpointID,
	}
	err := d.Leave(lr)
	assert.Nil(t, err)
}

func TestLeaveWithInvalidNetworkId(t *testing.T) {
	_, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	lr := &pluginNet.LeaveRequest{
		NetworkID:  "",
		EndpointID: r.EndpointID,
	}
	err := d.Leave(lr)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "invalid network id")
}

func TestLeaveWithInvalidEndpointId(t *testing.T) {
	_, d, r, ep := initEndpointData()
	d.networks[r.NetworkID].endpoints[r.EndpointID] = ep
	lr := &pluginNet.LeaveRequest{
		NetworkID:  r.EndpointID,
		EndpointID: "",
	}
	err := d.Leave(lr)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "invalid endpoint id")
}
