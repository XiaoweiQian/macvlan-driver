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
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.NotEmpty(t, res.InterfaceName)
	assert.NotEmpty(t, ep.srcName)
	if link, err := ns.NlHandle().LinkByName(ep.srcName); err == nil {
		ns.NlHandle().LinkDel(link)
	}
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
