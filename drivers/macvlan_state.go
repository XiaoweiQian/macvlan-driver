package drivers

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/netlabel"
	"github.com/docker/libnetwork/osl"
	"github.com/docker/libnetwork/types"
)

func (d *Driver) addNetwork(n *network) {
	d.Lock()
	d.networks[n.id] = n
	d.Unlock()
}

func (d *Driver) deleteNetwork(nid string) {
	d.Lock()
	delete(d.networks, nid)
	d.Unlock()
}

// getnetworks Safely returns a slice of existing Networks
func (d *Driver) getnetworks() []*network {
	d.Lock()
	defer d.Unlock()

	ls := make([]*network, 0, len(d.networks))
	for _, nw := range d.networks {
		ls = append(ls, nw)
	}

	return ls
}

func (n *network) endpoint(eid string) *endpoint {
	n.Lock()
	defer n.Unlock()

	return n.endpoints[eid]
}

func (n *network) addEndpoint(ep *endpoint) {
	n.Lock()
	n.endpoints[ep.id] = ep
	n.Unlock()
}

func (n *network) deleteEndpoint(eid string) {
	n.Lock()
	delete(n.endpoints, eid)
	n.Unlock()
}

func (n *network) getEndpoint(eid string) (*endpoint, error) {
	n.Lock()
	defer n.Unlock()
	if eid == "" {
		return nil, fmt.Errorf("endpoint id %s not found", eid)
	}
	if ep, ok := n.endpoints[eid]; ok {
		return ep, nil
	}

	return nil, nil
}

func validateID(nid, eid string) error {
	if nid == "" {
		return fmt.Errorf("invalid network id")
	}
	if eid == "" {
		return fmt.Errorf("invalid endpoint id")
	}
	return nil
}

func (n *network) sandbox() osl.Sandbox {
	n.Lock()
	defer n.Unlock()

	return n.sbox
}

func (n *network) setSandbox(sbox osl.Sandbox) {
	n.Lock()
	n.sbox = sbox
	n.Unlock()
}

func (d *Driver) getNetwork(id string) (*network, error) {
	d.Lock()
	defer d.Unlock()
	if id == "" {
		return nil, types.BadRequestErrorf("invalid network id: %s", id)
	}
	if nw, ok := d.networks[id]; ok {
		return nw, nil
	}

	return nil, types.NotFoundErrorf("network not found: %s", id)
}

func (d *Driver) network(nid string) *network {
	d.Lock()
	n, ok := d.networks[nid]
	d.Unlock()
	if !ok {
		n = d.getNetworkFromSwarm(nid)
		if n != nil {
			d.Lock()
			d.networks[nid] = n
			d.Unlock()
		}
	}

	return n
}

func (d *Driver) getNetworkFromSwarm(nid string) *network {
	if d.client == nil {
		logrus.Errorf("Docker clinet is nil.")
		return nil
	}
	nw, err := d.client.NetworkInfo(nid)
	if err != nil {
		return nil
	}
	logrus.Infof("Network (%s)  found from swarm", nw)
	subnets := nw.IPAM.Config
	opts := nw.Options
	options := make(map[string]interface{})
	options[netlabel.GenericData] = opts
	// parse and validate the config and bind to networkConfiguration
	config, err := parseNetworkOptions(nid, options)
	if err != nil {
		logrus.Errorf("Swarm:Network (%s)  found, but parseNetworkOptions error %v", nw, err)
		return nil
	}
	if err := config.processIPAMFromSwarm(nid, subnets); err != nil {
		logrus.Errorf("Swarm:Network (%s)  found, but processIPAMFromSwarm error %v", nw, err)
		return nil
	}

	n := &network{
		id:        nid,
		driver:    d,
		endpoints: endpointTable{},
		config:    config,
	}

	logrus.Infof("restore Network (%s) from swarm", n)
	return n
}
