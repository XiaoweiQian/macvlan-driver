package drivers_test

import (
	"fmt"
	"testing"

	. "github.com/XiaoweiQian/macvlan-driver/drivers"
	"github.com/XiaoweiQian/macvlan-driver/drivers/mocks"
	pluginNet "github.com/docker/go-plugins-helpers/network"
	"github.com/stretchr/testify/assert"
)

func TestInitWithOK(t *testing.T) {
	ms := new(mocks.MacStore)
	d := &Driver{
		Networks: NetworkTable{},
		Store:    ms,
	}
	ms.On("InitStore", d).Return(nil)
	d, err := Init(ms)
	assert.Nil(t, err)
	assert.NotNil(t, d)
}

func TestInitWithErr(t *testing.T) {
	ms := new(mocks.MacStore)
	d := &Driver{
		Networks: NetworkTable{},
		Store:    ms,
	}
	ms.On("InitStore", d).Return(fmt.Errorf("error"))
	d, err := Init(ms)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Failure during init macvlan local store . Error: error")
	assert.Nil(t, d)
}

func TestGetCapabilities(t *testing.T) {
	d := &Driver{}
	cap, err := d.GetCapabilities()
	assert.Nil(t, err)
	assert.EqualValues(t, pluginNet.GlobalScope, cap.Scope)
}
