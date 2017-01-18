package netutils

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomMAC(t *testing.T) {
	mac1 := GenerateRandomMAC()
	mac2 := GenerateRandomMAC()
	assert.NotEqual(t, mac1.String(), mac2.String())
}

func TestGenerateMacFromIP(t *testing.T) {
	ip1, _, _ := net.ParseCIDR("192.168.1.2/24")
	ip2, _, _ := net.ParseCIDR("192.168.1.3/24")
	mac1 := GenerateMACFromIP(ip1)
	mac2 := GenerateMACFromIP(ip1)
	mac3 := GenerateMACFromIP(ip2)
	assert.Equal(t, mac1.String(), mac2.String())
	assert.NotEqual(t, mac1.String(), mac3.String())

}

func TestGenerateRandomName(t *testing.T) {
	prefix := "eth"
	size := 7
	name, err := GenerateRandomName(prefix, size)
	name1, _ := GenerateRandomName(prefix, size)
	assert.Nil(t, err)
	assert.Contains(t, name, prefix)
	assert.Equal(t, len(name), len(prefix)+size)
	assert.NotEqual(t, name, name1)
}

func TestGenerateIfaceName(t *testing.T) {
	prefix := "eth"
	size := 7
	eth1, err := GenerateIfaceName(prefix, size)
	eth2, _ := GenerateIfaceName(prefix, size)
	assert.Nil(t, err)
	assert.Not(t, eth1, eth2)
}

func TestCreateVethPair(t *testing.T) {
	name1 := "veth1"
	name2 := "veth2"
	err := CreateVethPair(name1, name2)
	assert.Nil(t, err)
	err = DeleteVethPair(name1, name2)
	assert.Nil(t, err)
}
