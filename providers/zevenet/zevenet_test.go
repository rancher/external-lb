package zevenet

import (
	"testing"

	"github.com/rancher/external-lb/model"
)

func TestEncodeServiceName(t *testing.T) {
	const orig = "consul_consul_7734d6f0-0079-415d-984b-a827238c2427_rancher"

	c1 := encodeServiceName(orig)
	c2 := decodeServiceName(c1)

	if c2 != orig {
		t.Fatal("Re-encode failed")
	}
}

func TestGetServiceName(t *testing.T) {
	const expect = "consul--U--consul--U--7734d6f0-0079-415d-984b-a827238c2427--U--rancher"

	n := getServiceName(&model.LBConfig{LBTargetPoolName: "consul_consul_7734d6f0-0079-415d-984b-a827238c2427_rancher"})

	if n != expect {
		t.Fatal("Failed to extract service name")
	}
}

func TestGetServiceNameEx(t *testing.T) {
	sn, env, suffix, err := getServiceNameEx(&model.LBConfig{LBTargetPoolName: "consul_consul_7734d6f0-0079-415d-984b-a827238c2427_rancher"})

	if err != nil {
		t.Fatal(err)
	}

	if sn != "consul--U--consul" {
		t.Fatal("Failed to extract service name")
	}

	if env != "7734d6f0-0079-415d-984b-a827238c2427" {
		t.Fatal("Failed to extract environment UUID")
	}

	if suffix != "rancher" {
		t.Fatal("Failed to extract suffix")
	}
}
