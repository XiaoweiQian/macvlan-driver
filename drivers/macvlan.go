package drivers

import (
	"fmt"
	"os"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/stringid"
	pluginNet "github.com/docker/go-plugins-helpers/network"
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
	swarmHost           = "http://localhost:6732"
)

var driverModeOpt = macvlanType + modeOpt // mode --option macvlan_mode

type endpointTable map[string]*endpoint

// networkTable ...
type networkTable map[string]*network

// Driver ...
type Driver struct {
	networks networkTable
	store    macStore
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
func Init(d *Driver) (*Driver, error) {
	if d == nil {
		d = &Driver{
			networks: networkTable{},
			store:    &MacvlanStore{},
		}
	}

	var err error
	host := os.Getenv("SWARM_HOST")
	if host == "" {
		host = swarmHost
	}
	d.client, err = docker.NewClient(host)
	if err != nil {
		str := fmt.Sprintf("Could not connect to swarm. Error: %v", err)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}

	if err = d.store.InitStore(d); err != nil {
		str := fmt.Sprintf("Failure during init macvlan local store: %v", err)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}

	return d, nil
}

// GetCapabilities for swarm return GlobalScope
func (d *Driver) GetCapabilities() (*pluginNet.CapabilitiesResponse, error) {
	logrus.Infof("GetCapabilities macvlan")
	cap := &pluginNet.CapabilitiesResponse{Scope: pluginNet.GlobalScope}
	return cap, nil
}

// AllocateNetwork ...
func (d *Driver) AllocateNetwork(r *pluginNet.AllocateNetworkRequest) (*pluginNet.AllocateNetworkResponse, error) {
	id := r.NetworkID
	opts := r.Options
	logrus.Infof("AllocateNetwork macvlan with networkID=%s,opts=%s", id, opts)
	ipV4Data := r.IPv4Data
	ipV6Data := r.IPv6Data
	if id == "" {
		str := "invalid network id for macvlan network"
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}

	// reject a null v4 network
	if len(ipV4Data) == 0 || ipV4Data[0].Pool == "0.0.0.0/0" {
		str := "ipv4 pool is empty"
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}

	options := make(map[string]interface{})
	options[netlabel.GenericData] = opts
	// parse and validate the config and bind to networkConfiguration
	config, err := parseNetworkOptions(id, options)
	if err != nil {
		str := fmt.Sprintf("CreateNetwork opts is invalid %s", options)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
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
		str := fmt.Sprintf("CreateNetwork ipV4Data is invalid %s", ipv4)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}

	// verify the macvlan mode from -o macvlan_mode option
	switch config.MacvlanMode {
	case "", modeBridge:
		// default to macvlan bridge mode if -o macvlan_mode is empty
		config.MacvlanMode = modeBridge
	case modePrivate:
		config.MacvlanMode = modePrivate
	case modePassthru:
		config.MacvlanMode = modePassthru
	case modeVepa:
		config.MacvlanMode = modeVepa
	default:
		str := fmt.Sprintf("requested macvlan mode '%s' is not valid, 'bridge' mode is the macvlan driver default", config.MacvlanMode)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}
	// loopback is not a valid parent link
	if config.Parent == "lo" {
		str := fmt.Sprintf("loopback interface is not a valid %s parent link", macvlanType)
		logrus.Errorf(str)
		return nil, fmt.Errorf(str)
	}

	networkList := d.getnetworks()
	for _, nw := range networkList {
		if config.Parent == nw.config.Parent {
			str := fmt.Sprintf("network %s is already using parent interface %s",
				getDummyName(stringid.TruncateID(nw.config.ID)), config.Parent)
			logrus.Errorf(str)
			return nil, fmt.Errorf(str)
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
	id := r.NetworkID
	logrus.Infof("FreeNetwork macvlan id=%s", id)
	if id == "" {
		str := "invalid network id passed while freeing macvlan network"
		logrus.Errorf(str)
		return fmt.Errorf(str)
	}

	d.Lock()
	_, ok := d.networks[id]
	d.Unlock()

	if !ok {
		logrus.Warnf("macvlan network with id %s not found", id)
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
