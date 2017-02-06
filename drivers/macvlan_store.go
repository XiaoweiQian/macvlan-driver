package drivers

import (
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libkv/store/boltdb"
	"github.com/docker/libnetwork/datastore"
)

//MacStore ...
type MacStore interface {
	InitStore(d *Driver) error
	PopulateEndpoints() error
	StoreUpdate(kvObject datastore.KVObject) error
	StoreDelete(kvObject datastore.KVObject) error
}

//MacvlanStore ...
type MacvlanStore struct {
	store  datastore.DataStore
	driver *Driver
}

// networkConfiguration for this driver's network specific configuration
type configuration struct {
	ID               string
	Mtu              int
	dbIndex          uint64
	dbExists         bool
	Internal         bool
	Parent           string
	MacvlanMode      string
	CreatedSlaveLink bool
	Ipv4Subnets      []*ipv4Subnet
	Ipv6Subnets      []*ipv6Subnet
}

type ipv4Subnet struct {
	SubnetIP string
	GwIP     string
}

type ipv6Subnet struct {
	SubnetIP string
	GwIP     string
}

// InitStore drivers are responsible for caching their own persistent state
func (ms *MacvlanStore) InitStore(d *Driver) error {
	// initiate the boltdb
	boltdb.Register()
	var err error
	ms.store, err = datastore.NewDataStore(datastore.LocalScope, nil)
	ms.driver = d
	if err != nil {
		return fmt.Errorf("could not init macvlan local store. Error: %s", err)
	}
	if err = ms.PopulateEndpoints(); err != nil {
		logrus.Debugf("Failure during macvlan endpoints populate: %v", err)
	}
	return nil
}

// PopulateEndpoints ...
func (ms *MacvlanStore) PopulateEndpoints() error {
	kvol, err := ms.store.List(datastore.Key(macvlanEndpointPrefix), &endpoint{})
	if err != nil && err != datastore.ErrKeyNotFound {
		return fmt.Errorf("failed to get macvlan endpoints from store: %v", err)
	}

	if err == datastore.ErrKeyNotFound {
		logrus.Debugf("There is no endpoints in the localStore for key (%s).", macvlanEndpointPrefix)
		return nil
	}

	for _, kvo := range kvol {
		ep := kvo.(*endpoint)
		n := ms.driver.network(ep.nid)
		if n == nil {
			logrus.Debugf("Network (%s) not found for restored macvlan endpoint (%s)", ep.nid[0:7], ep.id[0:7])
			logrus.Debugf("Deleting stale macvlan endpoint (%s) from store", ep.id[0:7])
			if err := ms.StoreDelete(ep); err != nil {
				logrus.Debugf("Failed to delete stale macvlan endpoint (%s) from store", ep.id[0:7])
			}
			continue
		}
		n.endpoints[ep.id] = ep
		logrus.Debugf("Endpoint (%s) restored to network (%s)", ep.id[0:7], ep.nid[0:7])
	}

	return nil
}

// StoreUpdate used to update persistent macvlan network records as they are created
func (ms *MacvlanStore) StoreUpdate(kvObject datastore.KVObject) error {
	if ms.store == nil {
		logrus.Warnf("macvlan store not initialized. kv object %s is not added to the store", datastore.Key(kvObject.Key()...))
		return nil
	}
	if err := ms.store.PutObjectAtomic(kvObject); err != nil {
		return fmt.Errorf("failed to update macvlan store for object type %T: %v", kvObject, err)
	}

	return nil
}

// StoreDelete used to delete macvlan records from persistent cache as they are deleted
func (ms *MacvlanStore) StoreDelete(kvObject datastore.KVObject) error {
	if ms.store == nil {
		logrus.Debugf("macvlan store not initialized. kv object %s is not deleted from store", datastore.Key(kvObject.Key()...))
		return nil
	}
retry:
	if err := ms.store.DeleteObjectAtomic(kvObject); err != nil {
		if err == datastore.ErrKeyModified {
			if err := ms.store.GetObject(datastore.Key(kvObject.Key()...), kvObject); err != nil {
				return fmt.Errorf("could not update the kvobject to latest when trying to delete: %v", err)
			}
			goto retry
		}
		return err
	}

	return nil
}

func (config *configuration) MarshalJSON() ([]byte, error) {
	nMap := make(map[string]interface{})
	nMap["ID"] = config.ID
	nMap["Mtu"] = config.Mtu
	nMap["Parent"] = config.Parent
	nMap["MacvlanMode"] = config.MacvlanMode
	nMap["Internal"] = config.Internal
	nMap["CreatedSubIface"] = config.CreatedSlaveLink
	if len(config.Ipv4Subnets) > 0 {
		iis, err := json.Marshal(config.Ipv4Subnets)
		if err != nil {
			return nil, err
		}
		nMap["Ipv4Subnets"] = string(iis)
	}
	if len(config.Ipv6Subnets) > 0 {
		iis, err := json.Marshal(config.Ipv6Subnets)
		if err != nil {
			return nil, err
		}
		nMap["Ipv6Subnets"] = string(iis)
	}

	return json.Marshal(nMap)
}

func (config *configuration) UnmarshalJSON(b []byte) error {
	var (
		err  error
		nMap map[string]interface{}
	)

	if err = json.Unmarshal(b, &nMap); err != nil {
		return err
	}
	config.ID = nMap["ID"].(string)
	config.Mtu = int(nMap["Mtu"].(float64))
	config.Parent = nMap["Parent"].(string)
	config.MacvlanMode = nMap["MacvlanMode"].(string)
	config.Internal = nMap["Internal"].(bool)
	config.CreatedSlaveLink = nMap["CreatedSubIface"].(bool)
	if v, ok := nMap["Ipv4Subnets"]; ok {
		if err := json.Unmarshal([]byte(v.(string)), &config.Ipv4Subnets); err != nil {
			return err
		}
	}
	if v, ok := nMap["Ipv6Subnets"]; ok {
		if err := json.Unmarshal([]byte(v.(string)), &config.Ipv6Subnets); err != nil {
			return err
		}
	}

	return nil
}
