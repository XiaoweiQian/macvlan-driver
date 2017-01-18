package drivers

import (
	"fmt"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/stringid"
	pluginNet "github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/datastore"
	"github.com/docker/libnetwork/netlabel"
	"github.com/docker/libnetwork/osl"
	docker "github.com/fsouza/go-dockerclient"
)

const (
	vethLen             = 7
	containerVethPrefix = "eth"
	vethPrefix          = "veth"
	macvlanType         = "macvlan"  // driver type name
	modePrivate         = "private"  // macvlan mode private
	modeVepa            = "vepa"     // macvlan mode vepa
	modeBridge          = "bridge"   // macvlan mode bridge
	modePassthru        = "passthru" // macvlan mode passthrough
	parentOpt           = "parent"   // parent interface -o parent
	modeOpt             = "_mode"    // macvlan mode ux opt suffix
	swarmEndpoint       = "http://localhost:6732"
)

var driverModeOpt = macvlanType + modeOpt // mode --option macvlan_mode

type endpointTable map[string]*endpoint

type networkTable map[string]*network

// Driver ...
type Driver struct {
	networks networkTable
	store    datastore.DataStore
	client   *docker.Client
	sync.Once
	sync.Mutex
}

type network struct {
	id        string
	sbox      osl.Sandbox
	endpoints endpointTable
	driver    *Driver
	config    *configuration
	sync.Mutex
}

// Init macvlan remote driver
func Init() (*Driver, error) {
	d := &Driver{
		networks: networkTable{},
	}
	var err error
	d.client, err = docker.NewClient(swarmEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not connect to swarm. Error: %s", err)
	}

	if err = d.initStore(); err != nil {
		logrus.Debugf("Failure during init macvlan local store : %v", err)
	}

	return d, nil
}

// GetCapabilities for swarm return GlobalScope
func (d *Driver) GetCapabilities() (*pluginNet.CapabilitiesResponse, error) {
	logrus.Debugf("GetCapabilities macvlan")
	cap := &pluginNet.CapabilitiesResponse{Scope: pluginNet.GlobalScope}
	return cap, nil
}

// AllocateNetwork ...
func (d *Driver) AllocateNetwork(r *pluginNet.AllocateNetworkRequest) (*pluginNet.AllocateNetworkResponse, error) {
	id := r.NetworkID
	opts := r.Options
	logrus.Debugf("AllocateNetwork macvlan with networkID=%s,opts=%s", id, opts)
	ipV4Data := r.IPv4Data
	ipV6Data := r.IPv6Data
	if id == "" {
		return nil, fmt.Errorf("invalid network id for macvlan network")
	}

	// reject a null v4 network
	if len(ipV4Data) == 0 || ipV4Data[0].Pool == "0.0.0.0/0" {
		return nil, fmt.Errorf("ipv4 pool is empty")
	}

	options := make(map[string]interface{})
	options[netlabel.GenericData] = opts
	// parse and validate the config and bind to networkConfiguration
	config, err := parseNetworkOptions(id, options)
	if err != nil {
		return nil, err
	}

	config.ID = id
	ipv4 := []*pluginNet.IPAMData{}
	ipv6 := []*pluginNet.IPAMData{}
	for _, ipd := range ipV4Data {
		ipv4 = append(ipv4, &ipd)
	}
	for _, ipd := range ipV6Data {
		ipv6 = append(ipv6, &ipd)
	}
	err = config.processIPAM(id, ipv4, ipv6)
	if err != nil {
		return nil, err
	}

	networkList := d.getNetworks()
	for _, nw := range networkList {
		if config.Parent == nw.config.Parent {
			return nil, fmt.Errorf("network %s is already using parent interface %s",
				getDummyName(stringid.TruncateID(nw.config.ID)), config.Parent)
		}
	}

	n := &network{
		id:     id,
		driver: d,
		config: config,
	}

	d.Lock()
	d.networks[id] = n
	d.Unlock()
	res := &pluginNet.AllocateNetworkResponse{Options: opts}

	return res, nil

}

// FreeNetwork ...
func (d *Driver) FreeNetwork(r *pluginNet.FreeNetworkRequest) error {
	logrus.Debugf("FreeNetwork macvlan")
	id := r.NetworkID
	if id == "" {
		return fmt.Errorf("invalid network id passed while freeing macvlan network")
	}

	d.Lock()
	_, ok := d.networks[id]
	d.Unlock()

	if !ok {
		logrus.Debugf("macvlan network with id %s not found", id)
		return nil
	}

	d.Lock()
	delete(d.networks, id)
	d.Unlock()

	return nil
}

// DiscoverNew ...
func (d *Driver) DiscoverNew(r *pluginNet.DiscoveryNotification) error {
	logrus.Debugf("DiscoverNew macvlan")
	return nil
}

// DiscoverDelete ...
func (d *Driver) DiscoverDelete(r *pluginNet.DiscoveryNotification) error {
	logrus.Debugf("DiscoverDelete macvlan")
	return nil
}

// ProgramExternalConnectivity ...
func (d *Driver) ProgramExternalConnectivity(r *pluginNet.ProgramExternalConnectivityRequest) error {
	logrus.Debugf("ProgramExternalConnectivity macvlan")
	return nil
}

// RevokeExternalConnectivity ...
func (d *Driver) RevokeExternalConnectivity(r *pluginNet.RevokeExternalConnectivityRequest) error {
	logrus.Debugf("RevokeExternalConnectivity macvlan")
	return nil
}
