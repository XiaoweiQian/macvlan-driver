package rest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gavv/httpexpect"
)

const image = "172.19.146.181:5000/nginx:latest"

func initService() map[string]interface{} {
	service := map[string]interface{}{
		"Name": "service_test",
		"TaskTemplate": map[string]interface{}{
			"ContainerSpec": map[string]interface{}{
				"Image": image,
				//"Mounts": []interface{}{
				//	map[string]interface{}{},
				//},
				"User":      "",
				"DNSConfig": map[string]interface{}{},
			},
			"Networks": []interface{}{
				map[string]interface{}{
					"Target": "macvlan_test",
				},
			},
			"LogDriver": map[string]interface{}{
				"Name": "json-file",
				"Options": map[string]interface{}{
					"max-file": "3",
					"max-size": "10M",
				},
			},
			"Resources": map[string]interface{}{
				"Limits": map[string]interface{}{
					"MemoryBytes": 104857600,
				},
				"Reservations": map[string]interface{}{},
			},
			"RestartPolicy": map[string]interface{}{
				"Condition":   "on-failure",
				"Delay":       10000000000,
				"MaxAttempts": 10,
			},
			"Placement": map[string]interface{}{},
		},
		"Mode": map[string]interface{}{
			"Replicated": map[string]interface{}{
				"Replicas": 3,
			},
		},
		"UpdateConfig": map[string]interface{}{
			"Delay":         30000000000,
			"Parallelism":   2,
			"FailureAction": "pause",
		},
		"EndpointSpec": map[string]interface{}{
			"Mode": "none",
		},
		"Labels": map[string]interface{}{},
	}
	return service
}

func TestCreateAndDeleteService(t *testing.T) {
	nid, _ := initNetwork(t)
	s := initService()
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + nid).Expect().Status(http.StatusNoContent).NoContent()
	}()
	obj := e.POST("/services/create").WithJSON(s).
		Expect().Status(http.StatusCreated).JSON().Object()
	id := obj.Value("ID").String().Raw()
	time.Sleep(2 * time.Second)
	defer func() {
		e.DELETE("/services/" + id).Expect().Status(http.StatusOK)
	}()
	res := e.GET("/services/" + id).Expect().Status(http.StatusOK).JSON().Object()
	res.Value("ID").String().Equal(id)
	filter := `filters={"service":["service_test"],"desired-state":["running"]}`
	e.GET("/tasks").WithQueryString(filter).Expect().Status(http.StatusOK).JSON().Array().Length().Equal(3)
}

func TestCreateServiceWithDuplicateName(t *testing.T) {
	nid, _ := initNetwork(t)
	s := initService()
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + nid).Expect().Status(http.StatusNoContent).NoContent()
	}()
	obj := e.POST("/services/create").WithJSON(s).
		Expect().Status(http.StatusCreated).JSON().Object()
	id := obj.Value("ID").String().Raw()
	defer func() {
		e.DELETE("/services/" + id).Expect().Status(http.StatusOK)
	}()
	e.POST("/services/create").WithJSON(s).Expect().Status(http.StatusConflict)

}

func TestCreateServiceWithVip(t *testing.T) {
	nid, _ := initNetwork(t)
	s := initService()
	s["EndpointSpec"].(map[string]interface{})["mode"] = "vip"
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + nid).Expect().Status(http.StatusNoContent).NoContent()
	}()
	obj := e.POST("/services/create").WithJSON(s).
		Expect().Status(http.StatusCreated).JSON().Object()
	id := obj.Value("ID").String().Raw()
	time.Sleep(3 * time.Second)
	defer func() {
		e.DELETE("/services/" + id).Expect().Status(http.StatusOK)
	}()
	res := e.GET("/services/" + id).Expect().Status(http.StatusOK).JSON().Object()
	res.Value("ID").String().Equal(id)
	res.Value("Endpoint").Object().ContainsKey("VirtualIPs")
	filter := `filters={"service":["service_test"],"desired-state":["running"]}`
	e.GET("/tasks").WithQueryString(filter).Expect().Status(http.StatusOK).JSON().Array().Length().Equal(3)
}

func TestCreateServiceWithGlobal(t *testing.T) {
	nid, _ := initNetwork(t)
	s := initService()
	s["Mode"] = map[string]interface{}{
		"Global": map[string]interface{}{},
	}
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + nid).Expect().Status(http.StatusNoContent).NoContent()
	}()
	obj := e.POST("/services/create").WithJSON(s).
		Expect().Status(http.StatusCreated).JSON().Object()
	id := obj.Value("ID").String().Raw()
	time.Sleep(2 * time.Second)
	defer func() {
		e.DELETE("/services/" + id).Expect().Status(http.StatusOK)
	}()
	res := e.GET("/services/" + id).Expect().Status(http.StatusOK).JSON().Object()
	res.Value("ID").String().Equal(id)
	filter := `filters={"service":["service_test"],"desired-state":["running"]}`
	e.GET("/tasks").WithQueryString(filter).Expect().Status(http.StatusOK).JSON().Array().Length().Equal(1)
}

func TestCreateServiceWith2Network(t *testing.T) {
	nid1, _ := initNetwork(t)
	nid2, _ := initNetwork2(t)
	s := initService()
	s["TaskTemplate"].(map[string]interface{})["Networks"] = []interface{}{
		map[string]interface{}{
			"Target": "macvlan_test",
		},
		map[string]interface{}{
			"Target": "macvlan_test2",
		},
	}
	e := httpexpect.New(t, url)
	defer func() {
		e.DELETE("/networks/" + nid1).Expect().Status(http.StatusNoContent).NoContent()
		e.DELETE("/networks/" + nid2).Expect().Status(http.StatusNoContent).NoContent()
	}()
	obj := e.POST("/services/create").WithJSON(s).
		Expect().Status(http.StatusCreated).JSON().Object()
	id := obj.Value("ID").String().Raw()
	time.Sleep(2 * time.Second)
	defer func() {
		e.DELETE("/services/" + id).Expect().Status(http.StatusOK)
	}()
	res := e.GET("/services/" + id).Expect().Status(http.StatusOK).JSON().Object()
	res.Value("ID").String().Equal(id)
	//spec := res.Raw()["Spec"].(map[string]interface{})
	//assert.Equal(t, len(spec["Networks"].([]interface{})), 2)
	filter := `filters={"service":["service_test"],"desired-state":["running"]}`
	e.GET("/tasks").WithQueryString(filter).Expect().Status(http.StatusOK).JSON().Array().Length().Equal(3)
}
