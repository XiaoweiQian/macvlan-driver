package drivers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshaJSONForConfig(t *testing.T) {
	_, d, r, _ := initEndpointData()
	c := d.networks[r.NetworkID].config
	b, err := c.MarshalJSON()
	assert.Nil(t, err)
	c1 := &configuration{}
	err1 := c1.UnmarshalJSON(b)
	assert.Nil(t, err1)
	assert.EqualValues(t, c1, c)
}
