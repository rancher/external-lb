package zevenet

import (
	"testing"

	"github.com/rancher/external-lb/model"
)

/*
10.4.2018 18:03:02time="2018-04-10T16:03:02Z" level=info msg="Updating LB config: LBConfig={Endpoint: consul(\\.konsorten\\.net)?, PoolName: consul_consul_7734d6f0-0079-415d-984b-a827238c2427_rancher, TargetPort: 8500, Targets: [(10.209.1.42:8500), (10.209.1.28:8500), (10.209.1.27:8500), ]}"
10.4.2018 18:03:02time="2018-04-10T16:03:02Z" level=error msg="Failed to add farm MainHTTP: Failed to create service on Zevenet loadbalancer: New service failed: Error, the service consul--U--consul--U--7734d6f0-0079-415d-984b-a827238c2427--U--rancher already exists."
10.4.2018 18:03:03time="2018-04-10T16:03:03Z" level=error msg="Failed to add farm MainHTTPS: Failed to create service on Zevenet loadbalancer: New service failed: Error, the service consul--U--consul--U--7734d6f0-0079-415d-984b-a827238c2427--U--rancher already exists."
10.4.2018 18:03:03time="2018-04-10T16:03:03Z" level=info msg="Updating LB config: LBConfig={Endpoint: zabbix(\\.konsorten\\.net)?, PoolName: webinterface_zabbix_7734d6f0-0079-415d-984b-a827238c2427_rancher, TargetPort: 10058, Targets: [(10.209.1.42:10058), ]}"
10.4.2018 18:03:03time="2018-04-10T16:03:03Z" level=error msg="Failed to add farm MainHTTP: Failed to create service on Zevenet loadbalancer: New service failed: Error, the service webinterface--U--zabbix--U--7734d6f0-0079-415d-984b-a827238c2427--U--rancher already exists."
10.4.2018 18:03:04time="2018-04-10T16:03:04Z" level=error msg="Failed to add farm MainHTTPS: Failed to create service on Zevenet loadbalancer: New service failed: Error, the service webinterface--U--zabbix--U--7734d6f0-0079-415d-984b-a827238c2427--U--rancher already exists."
*/

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
