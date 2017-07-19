package avi

import (
	"encoding/json"
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

var vsJson = `{
       "uri_path":"/api/virtualservice",
       "model_name":"virtualservice",
       "data":{
         "network_profile_name":"System-TCP-Proxy",
         "flow_dist":"LOAD_AWARE",
         "delay_fairness":false,
         "avi_allocated_vip":false,
         "scaleout_ecmp":false,
         "analytics_profile_name":"System-Analytics-Profile",
         "cloud_type":"CLOUD_NONE",
         "weight":1,
         "cloud_name":"%s",
         "avi_allocated_fip":false,
         "max_cps_per_client":0,
         "type":"VS_TYPE_NORMAL",
         "use_bridge_ip_as_vip":false,
         "ign_pool_net_reach":true,
         "east_west_placement":false,
         "limit_doser":false,
         "ssl_sess_cache_avg_size":1024,
         "enable_autogw":true,
         "auto_allocate_ip":true,
         "enabled":true,
         "analytics_policy":{
           "client_insights":"ACTIVE",
           "metrics_realtime_update":{
             "duration":60,
             "enabled":false},
           "full_client_logs":{
             "duration":30,
             "enabled":false},
           "client_log_filters":[],
           "client_insights_sampling":{}
         },
         "vs_datascripts":[],
         "application_profile_ref":"%s",
	 "auto_allocate_ip": true,
         "name":"%s",
	 "dns_info": [{"fqdn": "%s"}],
	 "network_ref": "%s",
         "pool_ref":"%s",`

// checks if pool exists: returns the pool, else some error
func (p *AviProvider) CheckPoolExists(poolName string) (bool, AviRawDataType, error) {
	var resp AviRawDataType

	cresp, err := p.aviSession.GetCollection("/api/pool?name=" + poolName)
	if err != nil {
		log().Infof("Avi PoolExists check failed: %v", cresp)
		return false, resp, err
	}

	if cresp.Count == 0 {
		return false, resp, nil
	}
	nres, err := ConvertAviResponseToMapInterface(cresp.Results[0])
	if err != nil {
		return true, resp, err
	}
	return true, nres.(AviRawDataType), nil
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

func (p *AviProvider) GetResourceByName(resource, objname string) (AviRawDataType, error) {
	resp := make(AviRawDataType)
	res, err := p.aviSession.GetCollection("/api/" + resource + "?name=" + objname)
	if err != nil {
		log().Infof("Avi object exists check (res: %s, name: %s) failed: %v", resource, objname, res)
		return resp, err
	}

	if res.Count == 0 {
		return resp, fmt.Errorf("Resource name %s of type %s does not exist on the Avi Controller",
			objname, resource)
	}
	nres, err := ConvertAviResponseToMapInterface(res.Results[0])
	if err != nil {
		log().Infof("Resource unmarshal failed: %v", string(res.Results[0]))
		return resp, err
	}
	return nres.(AviRawDataType), nil
}

func (p *AviProvider) EnsurePoolExists(poolName string) (AviRawDataType, error) {
	exists, resp, err := p.CheckPoolExists(poolName)
	if exists {
		log().Infof("Pool %s already exists", poolName)
	}

	if exists || err != nil {
		return resp, err
	}

	return p.CreatePool(poolName)
}

func getPoolMembers(pool interface{}) []AviRawDataType {
	servers := make([]AviRawDataType, 0)
	pooldict := pool.(AviRawDataType)
	if pooldict["servers"] == nil {
		return servers
	}
	_servers := pooldict["servers"].([]interface{})
	for _, server := range _servers {
		servers = append(servers, server.(AviRawDataType))
	}

	// defport := int(pooldict["default_server_port"].(float64))
	// for _, server := range servers {
	// if server["port"] == nil {
	// server["port"] = defport
	//}
	// }

	return servers
}

func (p *AviProvider) RemovePoolMembers(pool AviRawDataType, deletedTasks dockerTasks) error {
	poolName := pool["name"].(string)
	currMembers := getPoolMembers(pool)
	retained := make([]interface{}, 0)
	for _, server := range currMembers {
		ip := server["ip"].(AviRawDataType)
		ipAddr := ip["addr"].(string)
		port := strconv.FormatInt(int64(server["port"].(float64)), 10)
		key := makeKey(ipAddr, port)
		if _, ok := deletedTasks[key]; ok {
			// this is deleted
			log().Debugf("Deleting pool member with key %s", key)
		} else {
			retained = append(retained, server)
		}
	}

	if len(currMembers) == len(retained) {
		log().Infof("Given members don't exist in pool %s; nothing to remove from pool", poolName)
		return nil
	}

	poolUuid := pool["uuid"].(string)
	pool["servers"] = retained
	log().Debugf("pool after assignment: %s", pool)
	res, err := p.aviSession.Put("/api/pool/"+poolUuid, pool)
	if err != nil {
		log().Infof("Avi update Pool failed: %v", res)
		return err
	}

	return nil
}

func (p *AviProvider) AddPoolMembers(pool AviRawDataType, addedTasks dockerTasks) error {
	// add new server to pool
	poolName := pool["name"].(string)
	poolUuid := pool["uuid"].(string)
	currMembers := getPoolMembers(pool)
	for _, member := range currMembers {
		port := strconv.FormatInt(int64(member["port"].(float64)), 10)
		ip := member["ip"].(AviRawDataType)
		ipAddr := ip["addr"].(string)
		key := makeKey(ipAddr, port)
		if _, ok := addedTasks[key]; ok {
			// already exists; remove
			delete(addedTasks, key)
		}
	}

	if len(addedTasks) == 0 {
		log().Infof("Pool %s has all intended members, no new member to be added.", poolName)
		return nil
	}

	for _, dt := range addedTasks {
		server := make(AviRawDataType)
		ip := make(AviRawDataType)
		ip["type"] = "V4"
		ip["addr"] = dt.ipAddr
		server["ip"] = ip
		server["port"] = dt.publicPort
		currMembers = append(currMembers, server)
		log().Debugf("currMembers in loop: %s", currMembers)
	}

	pool["servers"] = currMembers
	log().Debugf("pool after assignment: %s", pool)
	res, err := p.aviSession.Put("/api/pool/"+poolUuid, pool)
	if err != nil {
		log().Infof("Avi update Pool failed: %v", res)
		return err
	}

	return nil
}

// deletePool delete the named pool from Avi.
func (p *AviProvider) DeletePool(poolName string) error {
	exists, pool, err := p.CheckPoolExists(poolName)
	if err != nil || !exists {
		log().Infof("pool does not exist or can't obtain!: %v", pool)
		return err
	}
	poolUuid := pool["uuid"].(string)

	res, err := p.aviSession.Delete("/api/pool/" + poolUuid)
	if err != nil {
		log().Infof("Error deleting pool %s: %v", poolName, res)
		return err
	}

	return nil
}

func (p *AviProvider) GetPool(url string) (AviRawDataType, error) {
	resp := make(AviRawDataType)
	res, err := p.aviSession.GetCollection(url)
	if err != nil {
		log().Infof("Avi Pool Exists check failed: %v", res)
		return resp, err
	}

	if res.Count == 0 {
		return resp, fmt.Errorf("Pool [%s] does not exist on the Avi Controller",
			url)
	}
	nres, err := ConvertAviResponseToMapInterface(res.Results[0])
	if err != nil {
		log().Infof("VS unmarshal failed: %v", string(res.Results[0]))
		return resp, err
	}
	return nres.(AviRawDataType), nil
}

func (p *AviProvider) GetVS(vsname string) (AviRawDataType, error) {
	resp := make(AviRawDataType)
	res, err := p.aviSession.GetCollection("/api/virtualservice?name=" + vsname)
	if err != nil {
		log().Infof("Avi VS Exists check failed: %v", res)
		return resp, err
	}

	if res.Count == 0 {
		return resp, fmt.Errorf("Virtual Service %s does not exist on the Avi Controller",
			vsname)
	}
	nres, err := ConvertAviResponseToMapInterface(res.Results[0])
	if err != nil {
		log().Infof("VS unmarshal failed: %v", string(res.Results[0]))
		return resp, err
	}
	return nres.(AviRawDataType), nil
}

func (p *AviProvider) GetAllVses() ([]AviRawDataType, error) {
	allVses := make([]AviRawDataType, 0)
	res, err := p.aviSession.GetCollection("/api/virtualservice")
	if err != nil {
		log().Infof("Get all VSes failed: %v", res)
		return allVses, err
	}

	if res.Count == 0 {
		return allVses, nil
	}

	for i := 0; i < res.Count; i++ {
		nres, err := ConvertAviResponseToMapInterface(res.Results[i])
		if err != nil {
			log().Infof("VS unmarshal failed: %v", string(res.Results[i]))
		} else {
			allVses = append(allVses, nres.(AviRawDataType))
		}
	}

	return allVses, nil
}

func (p *AviProvider) CreatePool(poolName string) (AviRawDataType, error) {
	var resp AviRawDataType
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
		log().Infof("Error creating pool %s: %v", poolName, pres)
		return resp, err
	}

	return pres.(AviRawDataType), nil
}

func (p *AviProvider) AddCertificate() error {
	return nil
}

func (p *AviProvider) Create(vs *VS, tasks dockerTasks) error {
	log().Debugf("Creating pool %s for VS %s", vs.poolName, vs.name)
	pool, err := p.EnsurePoolExists(vs.poolName)
	if err != nil {
		return err
	}

	log().Debugf("Updating pool %s with members", vs.poolName)
	err = p.AddPoolMembers(pool, tasks)
	if err != nil {
		return err
	}

	if vs.sslEnabled {
		// add certificate
		err := p.AddCertificate()
		if err != nil {
			return err
		}
	}

	pvs, err := p.GetVS(vs.name)

	// TODO: Get the certs from Avi; remove following line
	sslCert := make(AviRawDataType)
	// for now, just mock an empty ref
	sslCert["url"] = ""
	// certName := ""

	if err == nil {
		// VS exists, check port etc
		servicePort := int(pvs["services"].([]interface{})[0].(AviRawDataType)["port"].(float64))
		if (vs.sslEnabled && servicePort == 443) ||
			(!vs.sslEnabled && servicePort == 80) {
			log().Infof("VS already exists %s", vs.name)
			return nil
		}

		// return another service exists with same name error
		return ErrDuplicateVS(vs.name)
	}

	appProfile, err := p.GetResourceByName("applicationprofile", vs.appProfileType)
	if err != nil {
		return err
	}

	// TODO: if you give networ, it asks for subnet. Fix later.
	nwRefUrl := ""
	// if p.cfg.AviIPAMNetwork != "" {
	// nwRef, err := p.GetResourceByName("network", p.cfg.AviIPAMNetwork)
	// if err != nil {
	// return err
	// }
	// nwRefUrl = nwRef["url"].(string)
	//}

	jsonstr := vsJson

	// TODO: For now, no ssl termination. Only enable ssl if port is
	// 443
	// if vs.sslEnabled {
	// jsonstr += `
	// "ssl_key_and_certificate_refs":[
	// "%s"
	// ],`
	// }

	jsonstr += `
         "services": [{"port": %s, "enable_ssl": %s}]
	    }
	}`

	//TODO: when supporting ssl termination; fix following which is
	// mocked above
	sslCertRef := sslCert["url"]
	if vs.sslEnabled {
		jsonstr = fmt.Sprintf(jsonstr,
			p.cfg.cloudName,
			appProfile["url"], vs.name, vs.fqdn,
			nwRefUrl, pool["url"], sslCertRef, "443", "true")
	} else {
		jsonstr = fmt.Sprintf(jsonstr,
			p.cfg.cloudName,
			appProfile["url"], vs.name, vs.fqdn,
			nwRefUrl, pool["url"], "80", "false")
	}

	var newVS interface{}
	json.Unmarshal([]byte(jsonstr), &newVS)
	log().Debugf("Sending request to create VS %s", vs.name)
	log().Debugf("DATA: %s", jsonstr)
	nres, err := p.aviSession.Post("api/macro", newVS)
	if err != nil {
		log().Infof("Failed creating VS: %s", vs.name)
		return err
	}

	log().Debugf("Created VS %s, response: %s", vs.name, nres)
	return nil
}

func (p *AviProvider) Delete(vs *VS) error {
	pvs, err := p.GetVS(vs.name)
	if err != nil {
		log().Warnf("Cloudn't retreive VS %s; error: %s", vs.name, err)
		return err
	}

	if pvs == nil {
		return nil
	}

	iresp, err := p.aviSession.Delete("/api/virtualservice/" + pvs["uuid"].(string))
	if err != nil {
		log().Warnf("Cloudn't delete VS %s; error: %s", vs.name, err)
		return err
	}

	log().Infof("VS delete response %s", iresp)

	// now delete the pool
	err = p.DeletePool(vs.poolName)
	if err != nil {
		log().Warnf("Cloudn't delete pool %s; error: %s", vs.poolName, err)
		return err
	}

	return nil
}

func (p *AviProvider) AddPoolMember(vs *VS, tasks dockerTasks) error {
	exists, pool, err := p.CheckPoolExists(vs.poolName)
	if err != nil {
		return err
	}
	if !exists {
		log().Warnf("Pool %s doesn't exist", vs.poolName)
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
