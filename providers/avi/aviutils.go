package avi

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rancher/external-lb/model"
)

const (
	POOL_ADD = iota
	POOL_REMOVE
	POOL_RECONCILE
)

func (p *AviProvider) updateVs(vs map[string]interface{}) error {
	vsUuid := vs["uuid"].(string)
	uri := "/api/virtualservice/" + vsUuid
	_, err := p.aviSession.Put(uri, vs)
	return err
}

func (p *AviProvider) updateVsMetadata(vs map[string]interface{}) error {
	log.Debugf("Updating service metadata for vs %s",
		vs["name"].(string))
	vs["service_metadata"] = p.cfg.lbSuffix
	return p.updateVs(vs)
}

func (p *AviProvider) checkExisitngPool(vs map[string]interface{},
	rnchrPoolName string) (map[string]interface{}, error) {
	empty := make(map[string]interface{})
	poolUrl := vs["pool_ref"].(string)
	u, err := url.Parse(poolUrl)
	if err != nil {
		return empty, fmt.Errorf("Invlid pool ref [%s]", poolUrl)
	}

	aviPool, err := p.GetPool(u.Path)
	if err != nil {
		return empty, err
	}

	aviPoolName := aviPool["name"].(string)
	if aviPoolName == rnchrPoolName {
		return aviPool, nil
	}

	if strings.HasSuffix(aviPoolName, p.cfg.lbSuffix) {
		vsName := vs["name"].(string)
		svcName := SvcNameFromRnchrPoolName(aviPoolName)
		// go p.RaiseDuplicateLabelEvent(vsName, svcName)
		err := fmt.Errorf("Lable/VS %s already used by service %s",
			vsName, svcName)
		return empty, err
	}

	// overwrite the pool name to match with what Rancher provides
	aviPool["name"] = rnchrPoolName
	return aviPool, nil
}

func (p *AviProvider) ensureVsHasPool(vs map[string]interface{},
	poolName string) (map[string]interface{}, error) {
	empty := make(map[string]interface{})
	if _, ok := vs["pool_ref"]; !ok {
		// pool doesn't exist; create one
		pool, err := p.EnsurePoolExists(poolName)
		if err != nil {
			return empty, err
		}

		vs["pool_ref"] = pool["url"]
		err = p.updateVs(vs)
		if err != nil {
			return empty, err
		}

		return pool, nil
	}

	return p.checkExisitngPool(vs, poolName)
}

func (p *AviProvider) convergePoolMembers(pool map[string]interface{},
	config model.LBConfig, op int) error {
	vsName := config.LBEndpoint
	dockerTasks := NewDockerTasks()
	for _, host := range config.LBTargets {
		hostPort, _ := strconv.Atoi(host.Port)
		dt := NewDockerTask(vsName, "tcp", host.HostIP, hostPort, -1)
		dockerTasks[dt.Key()] = dt
	}

	var err error
	switch op {
	case POOL_ADD:
		err = p.AddPoolMembers(pool, dockerTasks)
	case POOL_REMOVE:
		err = p.RemovePoolMembers(pool, dockerTasks)
	case POOL_RECONCILE:
		err = p.UpdatePoolMembers(pool, dockerTasks)
	}

	return err
}

func formLBConfig(vs map[string]interface{},
	pool map[string]interface{}) model.LBConfig {
	lbTargets := make([]model.LBTarget, 0)
	defaultPort := ""
	if _, ok := pool["default_server_port"]; ok {
		defaultPort = strconv.FormatInt(int64(pool["default_server_port"].(float64)), 10)
	}

	currMembers := getPoolMembers(pool)
	for _, server := range currMembers {
		ip := server["ip"].(map[string]interface{})
		ipAddr := ip["addr"].(string)
		targetPort := defaultPort
		if _, ok := server["port"]; ok {
			targetPort = strconv.FormatInt(int64(server["port"].(float64)), 10)
		}
		lbTargets = append(lbTargets, model.LBTarget{HostIP: ipAddr, Port: targetPort})
	}

	vsName := vs["name"].(string)
	poolName := pool["name"].(string)
	return model.LBConfig{LBEndpoint: vsName, LBTargetPoolName: poolName, LBTargetPort: defaultPort, LBTargets: lbTargets}
}

func GetVsFqdn(vs map[string]interface{}) (string, error) {
	data := vs
	if _, ok := vs["data"]; ok {
		data = vs["data"].(map[string]interface{})
	}

	_, hasFQDN := data["fqdn"]
	if hasFQDN {
		fqdn := data["fqdn"].(string)
		if fqdn != "" {
			return fqdn, nil
		}
	}

	_, hasDnsInfo := data["dns_info"]
	if !hasDnsInfo {
		err := fmt.Errorf("DNS Info not found in VS response %s", vs)
		return "", err
	}

	dnsInfo := data["dns_info"].([]interface{})[0].(map[string]interface{})
	return dnsInfo["fqdn"].(string), nil
}

func SvcNameFromRnchrPoolName(pName string) string {
	const sep = "_"
	return strings.Split(pName, sep)[0]
}

func VsFromCloud(vs map[string]interface{}, cloudRef string) bool {
	vsCloudRef, ok := vs["cloud_ref"]
	if ok && vsCloudRef.(string) == cloudRef {
		// VS not part of this cloud
		return true
	}

	return false
}

func VsHasMetadata(vs map[string]interface{}, metadata string) bool {
	svcMeta, ok := vs["service_metadata"]
	if ok && svcMeta.(string) == metadata {
		return true
	}

	return false
}

func (p *AviProvider) IsAssociatedVs(vs map[string]interface{}) bool {
	if VsFromCloud(vs, p.cloudRef) &&
		VsHasMetadata(vs, p.cfg.lbSuffix) {
		return true
	}

	return false
}

//func (p *AviProvider) addNewMembersToPool(pool map[string]interface{},
//	config model.LBConfig) error {
//	vsName := config.LBEndpoint
//	dockerTasks := NewDockerTasks()
//	for _, host := range config.LBTargets {
//		hostPort, _ := strconv.Atoi(host.Port)
//		dt := NewDockerTask(vsName, "tcp", host.HostIP, hostPort, -1)
//		dockerTasks[dt.Key()] = dt
//	}
//
//	err := p.AddPoolMembers(pool, dockerTasks)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func (p *AviProvider) removeMembersFromPool(pool map[string]interface{},
//	config model.LBConfig) error {
//	vsName := config.LBEndpoint
//	dockerTasks := NewDockerTasks()
//	for _, host := range config.LBTargets {
//		hostPort, _ := strconv.Atoi(host.Port)
//		dt := NewDockerTask(vsName, "tcp", host.HostIP, hostPort, -1)
//		dockerTasks[dt.Key()] = dt
//	}
//
//	err := p.RemovePoolMembers(pool, dockerTasks)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func (p *AviProvider) updatePoolMembers(pool map[string]interface{},
//	config model.LBConfig) error {
//	vsName := config.LBEndpoint
//	dockerTasks := NewDockerTasks()
//	for _, host := range config.LBTargets {
//		hostPort, _ := strconv.Atoi(host.Port)
//		dt := NewDockerTask(vsName, "tcp", host.HostIP, hostPort, -1)
//		dockerTasks[dt.Key()] = dt
//	}
//
//	err := p.UpdatePoolMembers(pool, dockerTasks)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
