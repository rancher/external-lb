package f5

import (
	"fmt"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/external-lb/providers"
	"github.com/scottdware/go-bigip"
)

const (
	ProviderName = "F5 BigIP"
	ProviderSlug = "f5_BigIP"
)

type F5BigIPProvider struct {
	client *bigip.BigIP
}

func init() {
	providers.RegisterProvider(ProviderSlug, new(F5BigIPProvider))
}

func (p *F5BigIPProvider) Init() error {
	f5_host := os.Getenv("F5_BIGIP_HOST")
	if len(f5_host) == 0 {
		return fmt.Errorf("F5_BIGIP_HOST is not set")
	}
	f5_admin := os.Getenv("F5_BIGIP_USER")
	if len(f5_admin) == 0 {
		return fmt.Errorf("F5_BIGIP_USER is not set")
	}
	f5_pwd := os.Getenv("F5_BIGIP_PWD")
	if len(f5_pwd) == 0 {
		return fmt.Errorf("F5_BIGIP_PWD is not set")
	}

	logrus.Debugf("Initializing f5 provider with host: %s, admin: %s, pwd-length: %d",
		f5_host, f5_admin, len(f5_pwd))

	p.client = bigip.NewSession(f5_host, f5_admin, f5_pwd, nil)

	if err := p.HealthCheck(); err != nil {
		return fmt.Errorf("Could not connect to f5 host '%s': %v", f5_host, err)
	}

	logrus.Infof("Configured %s provider using host %s", p.GetName(), f5_host)
	return nil
}

func (p *F5BigIPProvider) GetName() string {
	return ProviderName
}

func (p *F5BigIPProvider) HealthCheck() error {
	_, err := p.client.Pools()
	if err != nil {
		return fmt.Errorf("Failed to list f5 pools: %v", err)
	}
	return nil
}

func (p *F5BigIPProvider) AddLBConfig(config model.LBConfig) (string, error) {
	vServer, err := p.client.GetVirtualServer(config.LBEndpoint)
	if err != nil || vServer == nil {
		logrus.Errorf("f5 AddLBConfig: Error getting f5 virtual server, cannot add the config: %v\n", err)
		return "", err
	} else {
		//virtualserver exists, add nodes and pool

		nodes := config.LBTargets
		for _, node := range nodes {
			if p.nodeExists(node.HostIP, node.HostIP) {
				continue
			}
			//node does not exist, create new node
			err = p.client.CreateNode(node.HostIP, node.HostIP)
			if err != nil {
				logrus.Errorf("f5 AddLBConfig: Error creating node on f5: %v\n", err)
				return "", err
			} else {
				logrus.Debugf("f5 AddLBConfig: Success creating node %s", node.HostIP)
			}
		}

		//Create our pool if does not exist
		poolName := config.LBTargetPoolName
		if !p.poolExists(poolName) {
			err = p.client.CreatePool(poolName)
			if err != nil {
				logrus.Errorf("f5 AddLBConfig: Error creating pool: %s , err: %v\n", poolName, err)
				return "", err
			}
		}

		var pool *bigip.Pool
		pool, err = p.client.GetPool(poolName)
		if err != nil {
			logrus.Errorf("f5 AddLBConfig: Error getting back the pool: %v\n", err)
			return "", err
		} else {
			pool.AllowNAT = "yes"
			pool.AllowSNAT = "yes"
			err = p.client.ModifyPool(poolName, pool)
			if err != nil {
				logrus.Errorf("f5 AddLBConfig: Error modifying the pool: %v\n", err)
				return "", err
			}
		}

		// Add members to our pool if not already present

		poolMembers, err := p.client.PoolMembers(poolName)
		if err != nil {
			logrus.Errorf("f5 AddLBConfig: Error listing members of  pool: %v\n", err)
			return "", err
		}
		for _, node := range nodes {
			if !poolMemberExists(poolMembers, node.HostIP+":"+node.Port) {
				err = p.client.AddPoolMember(poolName, node.HostIP+":"+node.Port)
				if err != nil {
					logrus.Errorf("f5 AddLBConfig: Error adding member to pool: %v\n", err)
					return "", err
				}
			}
		}

		//Add pool to virtualserver provided
		updatedVs := bigip.VirtualServer{}
		updatedVs.Pool = poolName

		err = p.client.ModifyVirtualServer(config.LBEndpoint, &updatedVs)
		if err != nil {
			logrus.Errorf("f5 AddLBConfig: Error modifying virtual server: %v\n", err)
			return "", err
		} else {
			logrus.Debugf("f5 AddLBConfig: Success adding the LB config to f5")
		}

	}

	return "", nil
}

func (p *F5BigIPProvider) nodeExists(name string, nodeIp string) bool {
	bigIpNode, err := p.client.GetNode(name)
	if err != nil {
		logrus.Errorf("f5: Error getting f5 node: %v\n", err)
		return false
	}
	if bigIpNode != nil && bigIpNode.Address == nodeIp {
		return true
	}

	return false
}

func (p *F5BigIPProvider) poolExists(name string) bool {
	bigIpPool, err := p.client.GetPool(name)
	if err != nil {
		logrus.Errorf("f5: Error getting f5 pool: %v\n", err)
		return false
	}

	if bigIpPool != nil {
		return true
	}

	return false
}

func poolMemberExists(p *bigip.PoolMembers, member string) bool {
	for _, a := range p.PoolMembers {
		if a.Name == member {
			return true
		}
	}
	return false
}

//delete the LBConfig (unassign pool from virtualServer, remove pool, remove nodes)
func (p *F5BigIPProvider) RemoveLBConfig(config model.LBConfig) error {
	_, err := p.client.GetVirtualServer(config.LBEndpoint)
	if err != nil {
		logrus.Errorf("f5 RemoveLBConfig: Error getting f5 virtual server: %v\n", err)
		return err
	}
	//virtualserver exists,
	//Remove pool from virtualserver provided
	updatedVs := bigip.VirtualServer{}
	updatedVs.Pool = "None"

	err = p.client.ModifyVirtualServer(config.LBEndpoint, &updatedVs)

	if err != nil {
		logrus.Errorf("f5 RemoveLBConfig: Error modifying virtual server: %v\n", err)
		return err
	}

	poolName := config.LBTargetPoolName

	poolMembers, err := p.client.PoolMembers(poolName)
	var nodes []model.LBTarget
	if err != nil {
		logrus.Errorf("f5 RemoveLBConfig: Error listing pool members for pool: %s, err: %v\n", poolName, err)
	} else {
		for _, member := range poolMembers.PoolMembers {
			nodeParts := strings.Split(member.Name, ":")
			if len(nodeParts) == 2 {
				node := model.LBTarget{}
				node.HostIP = nodeParts[0]
				node.Port = nodeParts[1]
				nodes = append(nodes, node)
			}
		}
	}
	//remove the pool
	err = p.client.DeletePool(poolName)
	if err != nil {
		logrus.Errorf("f5 RemoveLBConfig: Error removing pool: %s , err: %v\n", poolName, err)
	}
	//remove the nodes under the pool
	for _, node := range nodes {
		if p.nodeExists(node.HostIP, node.HostIP) {
			//node exist, delete node
			err = p.client.DeleteNode(node.HostIP)
			if err != nil {
				logrus.Errorf("f5 RemoveLBConfig: Error removing node on f5: %v\n", err)
			}
		}
	}

	logrus.Debugf("f5 RemoveLBConfig: Success")
	return nil
}

func (p *F5BigIPProvider) UpdateLBConfig(config model.LBConfig) (string, error) {
	err := p.RemoveLBConfig(config)
	if err != nil {
		logrus.Errorf("f5 UpdateLBConfig: Error removing existing config: %v\n", err)
		return "", err
	}

	_, err = p.AddLBConfig(config)
	if err != nil {
		logrus.Errorf("f5 UpdateLBConfig: Error adding back the config: %v\n", err)
		return "", err
	}

	logrus.Debugf("f5 UpdateLBConfig: Success")
	return "", nil
}

func (p *F5BigIPProvider) GetLBConfigs() ([]model.LBConfig, error) {
	//list all virtualServers
	// for each vs -> LBEndpoint
	// get the pool -> LBTargetPoolName
	// pool members -> LB Targets hostIP : Port
	var lbConfigs []model.LBConfig

	vServers, err := p.client.VirtualServers()
	if err != nil {
		logrus.Errorf("f5 GetLBConfigs: Error listing f5 virtual servers: %v\n", err)
		return lbConfigs, err
	}

	for _, vServer := range vServers.VirtualServers {
		if vServer.Pool != "" {
			pool, err := p.client.GetPool(strings.TrimPrefix(vServer.Pool, "/Common/"))
			if err != nil {
				logrus.Errorf("f5 GetLBConfigs: Error getting the pool: %s, err: %v\n", vServer.Pool, err)
				continue
			}
			lbConfig := model.LBConfig{}
			lbConfig.LBEndpoint = vServer.Name
			lbConfig.LBTargetPoolName = pool.Name

			var nodes []model.LBTarget

			poolMembers, err := p.client.PoolMembers(pool.Name)
			if err != nil {
				logrus.Errorf("f5 GetLBConfigs: Error listing pool members for pool: %s, err: %v\n", pool.Name, err)
			} else {
				for _, member := range poolMembers.PoolMembers {
					nodeParts := strings.Split(member.Name, ":")
					if len(nodeParts) == 2 {
						node := model.LBTarget{}
						node.HostIP = nodeParts[0]
						node.Port = nodeParts[1]
						nodes = append(nodes, node)
					}
				}
			}

			lbConfig.LBTargets = nodes
			lbConfigs = append(lbConfigs, lbConfig)
		}
	}

	logrus.Debugf("f5 GetLBConfigs returned: %v\n", lbConfigs)

	return lbConfigs, nil

}
