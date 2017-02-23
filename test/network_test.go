package rest_test

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect"
)

const url = "http://127.0.0.1:6732/v1.25"

func TestNetwork(t *testing.T) {
	e := httpexpect.New(t, url)
	net := map[string]interface{}{
		"Name":           "macvlan_test",
		"CheckDuplicate": true,
		"Driver":         "macvlan_swarm",
		"EnableIPv6":     true,
		"IPAM": map[string]interface{}{
			"Driver": "default",
			"Config": []interface{}{
				map[string]interface{}{
					"Subnet":  "172.20.0.0/16",
					"IPRange": "172.20.10.0/24",
					"Gateway": "172.20.10.11",
				},
			},
			"Options": map[string]interface{}{},
		},
		"Internal": false,
		"Options": map[string]interface{}{
			"parent": "eth0.20",
		},
		"Labels": map[string]interface{}{},
	}
	obj := e.POST("/networks/create").WithJSON(net).
		Expect().Status(http.StatusCreated).JSON().Object()
	id := obj.Value("Id").String().Raw()
	res := e.GET("/networks/" + id).Expect().Status(http.StatusOK).JSON().Object()
	res.Value("Driver").String().Equal("macvlan_swarm")
	e.DELETE("/networks/" + id).Expect().Status(http.StatusNoContent).NoContent()
}
