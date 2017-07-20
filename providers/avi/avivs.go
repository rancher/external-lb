package avi

import (
	"fmt"
	"strconv"
)

const (
	APP_PROFILE_HTTPS = "System-Secure-HTTP"
	APP_PROFILE_HTTP  = "System-HTTP"
	APP_PROFILE_TCP   = "System-L4-Application"
)

type VS struct {
	name           string
	poolName       string
	sslEnabled     bool
	fqdn           string
	appProfileType string
}

type ErrDuplicateVS string

func (val ErrDuplicateVS) Error() string {
	return fmt.Sprintf("VS with name %v already exists", string(val))
}

func (val ErrDuplicateVS) String() string {
	return fmt.Sprintf("ErrDuplicateVS(%v)", string(val))
}

type ErrServerConnection string

func (val ErrServerConnection) Error() string {
	return fmt.Sprintf("Failed to connect to Avi Controller: %v", string(val))
}

func (val ErrServerConnection) String() string {
	return fmt.Sprintf("ErrServerConnection(%v)", string(val))
}

func InitAviSession(cfg *AviConfig) (*AviSession, error) {
	insecure := !cfg.sslVerify
	portStr := strconv.Itoa(cfg.controllerPort)
	netloc := cfg.controllerIpAddr + ":" + portStr // 10.0.1.4:9443 typish
	aviSession := NewAviSession(netloc,
		cfg.username,
		cfg.password,
		insecure)
	err := aviSession.InitiateSession()
	return aviSession, err
}

// checks if pool exists: returns the pool, else some error
func (p *AviProvider) CheckPoolExists(poolName string) (bool, map[string]interface{}, error) {
	var resp map[string]interface{}

	cresp, err := p.aviSession.GetCollection("/api/pool?name=" + poolName)
	if err != nil {
		log.Infof("Avi PoolExists check failed: %v", cresp)
		return false, resp, err
	}

	if cresp.Count == 0 {
		return false, resp, nil
	}
	nres, err := ConvertAviResponseToMapInterface(cresp.Results[0])
	if err != nil {
		return true, resp, err
	}
	return true, nres.(map[string]interface{}), nil
}

func (p *AviProvider) GetCloudRef() (string, error) {
	cloudName := p.cfg.cloudName
	if cloudName == "Default-Cloud" {
		return "", nil
	}
	cloud, err := p.GetResourceByName("cloud", cloudName)
	if err != nil {
		return "", err
	}
	return cloud["url"].(string), nil
}

func (p *AviProvider) GetResourceByName(resource, objname string) (map[string]interface{}, error) {
	resp := make(map[string]interface{})
	res, err := p.aviSession.GetCollection("/api/" + resource + "?name=" + objname)
	if err != nil {
		log.Infof("Avi object exists check (res: %s, name: %s) failed: %v", resource, objname, res)
		return resp, err
	}

	if res.Count == 0 {
		return resp, fmt.Errorf("Resource name %s of type %s does not exist on the Avi Controller",
			objname, resource)
	}
	nres, err := ConvertAviResponseToMapInterface(res.Results[0])
	if err != nil {
		log.Infof("Resource unmarshal failed: %v", string(res.Results[0]))
		return resp, err
	}
	return nres.(map[string]interface{}), nil
}

func (p *AviProvider) EnsurePoolExists(poolName string) (map[string]interface{}, error) {
	exists, resp, err := p.CheckPoolExists(poolName)
	if exists {
		log.Infof("Pool %s already exists", poolName)
	}

	if exists || err != nil {
		return resp, err
	}

	return p.CreatePool(poolName)
}

func getPoolMembers(pool interface{}) []map[string]interface{} {
	servers := make([]map[string]interface{}, 0)
	pooldict := pool.(map[string]interface{})
	if pooldict["servers"] == nil {
		return servers
	}
	_servers := pooldict["servers"].([]interface{})
	for _, server := range _servers {
		servers = append(servers, server.(map[string]interface{}))
	}

	// defport := int(pooldict["default_server_port"].(float64))
	// for _, server := range servers {
	// if server["port"] == nil {
	// server["port"] = defport
	//}
	// }

	return servers
}

func (p *AviProvider) UpdatePoolMembers(pool map[string]interface{},
	allTasks dockerTasks) error {
	poolName := pool["name"].(string)
	aviPoolMembers := getPoolMembers(pool)
	retained := make([]interface{}, 0)
	for _, server := range aviPoolMembers {
		ip := server["ip"].(map[string]interface{})
		ipAddr := ip["addr"].(string)
		port := strconv.FormatInt(int64(server["port"].(float64)), 10)
		key := makeKey(ipAddr, port)
		if _, ok := allTasks[key]; ok {
			// this is retained
			retained = append(retained, server)
			delete(allTasks, key)
		}
	}

	if len(allTasks) >= 1 {
		for _, dt := range allTasks {
			server := make(map[string]interface{})
			ip := make(map[string]interface{})
			ip["type"] = "V4"
			ip["addr"] = dt.ipAddr
			server["ip"] = ip
			server["port"] = dt.publicPort
			retained = append(retained, server)
		}
	}

	poolUuid := pool["uuid"].(string)
	pool["servers"] = retained
	log.Debugf("pool %s has updated members: %s", poolName, retained)
	res, err := p.aviSession.Put("/api/pool/"+poolUuid, pool)
	if err != nil {
		log.Infof("Avi update Pool failed: %v", res)
		return err
	}

	return nil
}

func (p *AviProvider) RemovePoolMembers(pool map[string]interface{}, deletedTasks dockerTasks) error {
	poolName := pool["name"].(string)
	currMembers := getPoolMembers(pool)
	retained := make([]interface{}, 0)
	for _, server := range currMembers {
		ip := server["ip"].(map[string]interface{})
		ipAddr := ip["addr"].(string)
		port := strconv.FormatInt(int64(server["port"].(float64)), 10)
		key := makeKey(ipAddr, port)
		if _, ok := deletedTasks[key]; ok {
			// this is deleted
			log.Debugf("Deleting pool member with key %s", key)
		} else {
			retained = append(retained, server)
		}
	}

	if len(currMembers) == len(retained) {
		log.Infof("Given members don't exist in pool %s; nothing to remove from pool", poolName)
		return nil
	}

	poolUuid := pool["uuid"].(string)
	pool["servers"] = retained
	log.Debugf("pool after assignment: %s", pool)
	res, err := p.aviSession.Put("/api/pool/"+poolUuid, pool)
	if err != nil {
		log.Infof("Avi update Pool failed: %v", res)
		return err
	}

	return nil
}

func (p *AviProvider) AddPoolMembers(pool map[string]interface{}, addedTasks dockerTasks) error {
	// add new server to pool
	poolName := pool["name"].(string)
	poolUuid := pool["uuid"].(string)
	currMembers := getPoolMembers(pool)
	for _, member := range currMembers {
		port := strconv.FormatInt(int64(member["port"].(float64)), 10)
		ip := member["ip"].(map[string]interface{})
		ipAddr := ip["addr"].(string)
		key := makeKey(ipAddr, port)
		if _, ok := addedTasks[key]; ok {
			// already exists; remove
			delete(addedTasks, key)
		}
	}

	if len(addedTasks) == 0 {
		log.Infof("Pool %s has all intended members, no new member to be added.", poolName)
		return nil
	}

	for _, dt := range addedTasks {
		server := make(map[string]interface{})
		ip := make(map[string]interface{})
		ip["type"] = "V4"
		ip["addr"] = dt.ipAddr
		server["ip"] = ip
		server["port"] = dt.publicPort
		currMembers = append(currMembers, server)
		log.Debugf("currMembers in loop: %s", currMembers)
	}

	pool["servers"] = currMembers
	log.Debugf("pool after assignment: %s", pool)
	res, err := p.aviSession.Put("/api/pool/"+poolUuid, pool)
	if err != nil {
		log.Infof("Avi update Pool failed: %v", res)
		return err
	}

	return nil
}

// deletePool delete the named pool from Avi.
func (p *AviProvider) DeletePool(poolName string) error {
	exists, pool, err := p.CheckPoolExists(poolName)
	if err != nil || !exists {
		log.Infof("pool does not exist or can't obtain!: %v", pool)
		return err
	}
	poolUuid := pool["uuid"].(string)

	res, err := p.aviSession.Delete("/api/pool/" + poolUuid)
	if err != nil {
		log.Infof("Error deleting pool %s: %v", poolName, res)
		return err
	}

	return nil
}

func (p *AviProvider) GetPool(url string) (map[string]interface{}, error) {
	resp := make(map[string]interface{})
	res, err := p.aviSession.Get(url)
	if err != nil {
		log.Infof("Avi Pool Exists check failed: %v", err)
		return resp, err
	}

	return res.(map[string]interface{}), nil
}

func (p *AviProvider) GetVS(vsname string) (map[string]interface{}, error) {
	resp := make(map[string]interface{})
	res, err := p.aviSession.GetCollection("/api/virtualservice?name=" + vsname)
	if err != nil {
		log.Infof("Avi VS Exists check failed: %v", err)
		return resp, err
	}

	if res.Count == 0 {
		return resp, fmt.Errorf("Virtual Service %s does not exist on the Avi Controller",
			vsname)
	}
	nres, err := ConvertAviResponseToMapInterface(res.Results[0])
	if err != nil {
		log.Infof("VS unmarshal failed: %v", string(res.Results[0]))
		return resp, err
	}
	return nres.(map[string]interface{}), nil
}

func (p *AviProvider) GetAllVses() ([]map[string]interface{}, error) {
	allVses := make([]map[string]interface{}, 0)
	res, err := p.aviSession.GetCollection("/api/virtualservice")
	if err != nil {
		log.Infof("Get all VSes failed: %v", res)
		return allVses, err
	}

	if res.Count == 0 {
		return allVses, nil
	}

	for i := 0; i < res.Count; i++ {
		nres, err := ConvertAviResponseToMapInterface(res.Results[i])
		if err != nil {
			log.Infof("VS unmarshal failed: %v", string(res.Results[i]))
		} else {
			allVses = append(allVses, nres.(map[string]interface{}))
		}
	}

	return allVses, nil
}

func (p *AviProvider) CreatePool(poolName string) (map[string]interface{}, error) {
	var resp map[string]interface{}
	pool := make(map[string]string)
	pool["name"] = poolName
	cref, err := p.GetCloudRef()
	if err != nil {
		return resp, err
	}
	if cref != "" {
		pool["cloud_ref"] = cref
	}
	pres, err := p.aviSession.Post("/api/pool", pool)
	if err != nil {
		log.Infof("Error creating pool %s: %v", poolName, pres)
		return resp, err
	}

	return pres.(map[string]interface{}), nil
}

func (p *AviProvider) AddPoolMember(vs *VS, tasks dockerTasks) error {
	exists, pool, err := p.CheckPoolExists(vs.poolName)
	if err != nil {
		return err
	}
	if !exists {
		log.Warnf("Pool %s doesn't exist", vs.poolName)
		return nil
	}

	return p.AddPoolMembers(pool, tasks)
}

func (p *AviProvider) RemovePoolMember(vs *VS, tasks dockerTasks) error {
	exists, pool, err := p.CheckPoolExists(vs.poolName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	return p.RemovePoolMembers(pool, tasks)
}
