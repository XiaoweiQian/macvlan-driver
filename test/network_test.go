package rest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gavv/httpexpect"
)

const url = "http://localhost:6732/v1.25"

func initNetwork(t *testing.T) (string, map[string]interface{}) {
	time.Sleep(1 * time.Second)
	e := httpexpect.New(t, url)
	net := map[string]interface{}{
		"Name":           "macvlan_test",
		"CheckDuplicate": true,
		"Driver":         "macvlan_swarm",
		"EnableIPv6":     false,
		"IPAM": map[string]interface{}{
			"Driver": "default",
			"Config": []interface{}{
				map[string]interface{}{
					"Subnet": "192.168.10.0/24",
				},
			},
			"Options": map[string]interface{}{},
		},
		"Internal": false,
		"Options": map[string]interface{}{
			"parent":       "eth0.99",
			"macvlan_mode": "bridge",
		},
		"Labels": map[string]interface{}{},
	}
	obj := e.POST("/networks/create").WithJSON(net).
		Expect().Status(http.StatusCreated).JSON().Object()
	id := obj.Value("Id").String().Raw()
	time.Sleep(1 * time.Second)
	return id, net
}

func initNetwork2(t *testing.T) (string, map[string]interface{}) {
	time.Sleep(1 * time.Second)
	e := httpexpect.New(t, url)
	net := map[string]interface{}{
		"Name":           "macvlan_test2",
		"CheckDuplicate": true,
		"Driver":         "macvlan_swarm",
		"EnableIPv6":     false,
		"IPAM": map[string]interface{}{
			"Driver": "default",
			"Config": []interface{}{
				map[string]interface{}{
					"Subnet": "192.168.20.0/24",
				},
			},
			"Options": map[string]interface{}{},
		},
		"Internal": false,
		"Options": map[string]interface{}{
			"parent":       "eth0.100",
			"macvlan_mode": "bridge",
		},
		"Labels": map[string]interface{}{},
	}
	obj := e.POST("/networks/create").WithJSON(net).
		Expect().Status(http.StatusCreated).JSON().Object()
	id := obj.Value("Id").String().Raw()
	time.Sleep(1 * time.Second)
	return id, net
}

func TestCreateAndDeleteNetwork(t *testing.T) {
	id, _ := initNetwork(t)
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + id).Expect().Status(http.StatusNoContent).NoContent()
	}()
	res := e.GET("/networks/" + id).Expect().Status(http.StatusOK).JSON().Object()
	res.Value("Driver").String().Equal("macvlan_swarm")
}

func TestCreateNetworkWithDuplicateName(t *testing.T) {
	id, net := initNetwork(t)
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + id).Expect().Status(http.StatusNoContent).NoContent()
	}()
	e.POST("/networks/create").WithJSON(net).
		Expect().Status(http.StatusInternalServerError)
}

func TestCreateNetworkWithInvalidSubnet(t *testing.T) {
	id, net := initNetwork(t)
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + id).Expect().Status(http.StatusNoContent).NoContent()
	}()
	net["name"] = "macvlan_test1"
	ipam := net["IPAM"].(map[string]interface{})
	sub := ipam["Config"].([]interface{})[0].(map[string]interface{})
	sub["Subnet"] = ""
	e.POST("/networks/create").WithJSON(net).
		Expect().Status(http.StatusInternalServerError)
}

func TestCreateNetworkWithDuplicateParent(t *testing.T) {
	id, net := initNetwork(t)
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + id).Expect().Status(http.StatusNoContent).NoContent()
	}()
	net["name"] = "macvlan_test10"
	ipam := net["IPAM"].(map[string]interface{})
	sub := ipam["Config"].([]interface{})[0].(map[string]interface{})
	sub["Subnet"] = "192.168.1.0/24"
	e.POST("/networks/create").WithJSON(net).
		Expect().Status(http.StatusInternalServerError)
}

func TestCreateNetworkWithInvalidParent(t *testing.T) {
	id, net := initNetwork(t)
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + id).Expect().Status(http.StatusNoContent).NoContent()
	}()
	net["name"] = "macvlan_test1"
	ipam := net["IPAM"].(map[string]interface{})
	sub := ipam["Config"].([]interface{})[0].(map[string]interface{})
	sub["Subnet"] = "192.168.1.0/24"
	opts := net["Options"].(map[string]interface{})
	opts["parent"] = "eth0:20"
	e.POST("/networks/create").WithJSON(net).
		Expect().Status(http.StatusInternalServerError)
}
