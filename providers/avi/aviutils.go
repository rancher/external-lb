package avi

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/rancher/external-lb/model"
)

func (p *AviProvider) updateVs(vs map[string]interface{}) error {
	vsUuid := vs["uuid"].(string)
	uri := "/api/virtualservice?uuid=" + vsUuid
	_, err := p.aviSession.Put(uri, vs)
	return err
}

func (p *AviProvider) checkExisitngPool(vs map[string]interface{}, poolName string) (map[string]interface{}, error) {
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
	if aviPoolName != poolName {
		aviPool["name"] = poolName
	}

	return aviPool, nil
}

func (p *AviProvider) ensureVsHasPool(vs map[string]interface{}, poolName string) (map[string]interface{}, error) {
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

func (p *AviProvider) addNewMembersToPool(pool map[string]interface{}, config model.LBConfig) error {
	vsName := config.LBEndpoint
	dockerTasks := NewDockerTasks()
	for _, host := range config.LBTargets {
		hostPort, _ := strconv.Atoi(host.Port)
		dt := NewDockerTask(vsName, "tcp", host.HostIP, hostPort, -1)
		dockerTasks[dt.Key()] = dt
	}

	err := p.AddPoolMembers(pool, dockerTasks)
	if err != nil {
		return err
	}

	return nil
}

func (p *AviProvider) removeMembersFromPool(pool map[string]interface{}, config model.LBConfig) error {
	vsName := config.LBEndpoint
	dockerTasks := NewDockerTasks()
	for _, host := range config.LBTargets {
		hostPort, _ := strconv.Atoi(host.Port)
		dt := NewDockerTask(vsName, "tcp", host.HostIP, hostPort, -1)
		dockerTasks[dt.Key()] = dt
	}

	err := p.RemovePoolMembers(pool, dockerTasks)
	if err != nil {
		return err
	}

	return nil
}

func formLBConfig(vs map[string]interface{}, pool map[string]interface{}) model.LBConfig {
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
		lbTargets = append(lbTargets, model.LBTarget{ipAddr, targetPort})
	}

	vsName := vs["name"].(string)
	poolName := pool["name"].(string)
	return model.LBConfig{vsName, poolName, defaultPort, lbTargets}
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
		return "", fmt.Errorf("DNS Info not found in VS response %s", vs)
	}

	dnsInfo := data["dns_info"].([]interface{})[0].(map[string]interface{})
	return dnsInfo["fqdn"].(string), nil
}
