package f5

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/external-lb/providers"
	"github.com/scottdware/go-bigip"
	"os"
	"strings"
)

const (
	name = "f5_BigIP"
)

var (
	client *bigip.BigIP
)

func init() {
	f5_host := os.Getenv("F5_BIGIP_HOST")
	if len(f5_host) == 0 {
		logrus.Fatalf("F5_BIGIP_HOST is not set, skipping init of %s provider", name)
		return
	}
	f5_admin := os.Getenv("F5_BIGIP_USER")
	if len(f5_admin) == 0 {
		logrus.Fatalf("F5_BIGIP_USER is not set, skipping init of %s provider", name)
		return
	}
	f5_pwd := os.Getenv("F5_BIGIP_PWD")
	if len(f5_pwd) == 0 {
		logrus.Fatalf("F5_BIGIP_PWD is not set, skipping init of %s provider", name)
		return
	}

	client = bigip.NewSession(f5_host, f5_admin, f5_pwd)
	err := checkF5Connection()
	if err != nil {
		logrus.Fatalf("Connecting to f5 host %v does not work, error: %v", f5_host, err)
		return
	}

	f5BigIPHandler := &F5BigIPHandler{}
	if err := providers.RegisterProvider(name, f5BigIPHandler); err != nil {
		logrus.Fatalf("Could not register %s provider", name)
	}

	logrus.Infof("Configured %s LB provider", f5BigIPHandler.GetName())

}

type F5BigIPHandler struct {
}

func (*F5BigIPHandler) GetName() string {
	return name
}

func (*F5BigIPHandler) AddLBConfig(config model.LBConfig) error {

	vServer, err := client.GetVirtualServer(config.LBEndpoint)
	if err != nil || vServer == nil {
		logrus.Errorf("f5 AddLBConfig: Error getting f5 virtual server, cannot add the config: %v\n", err)
		return err
	} else {
		//virtualserver exists, add nodes and pool

		nodes := config.LBTargets
		for _, node := range nodes {
			if nodeExists(node.HostIP, node.HostIP) {
				continue
			}
			//node does not exist, create new node
			err = client.CreateNode(node.HostIP, node.HostIP)
			if err != nil {
				logrus.Errorf("f5 AddLBConfig: Error creating node on f5: %v\n", err)
				return err
			} else {
				logrus.Debugf("f5 AddLBConfig: Success creating node %s", node.HostIP)
			}
		}

		//Create our pool if does not exist
		poolName := config.LBTargetPoolName
		if !poolExists(poolName) {
			err = client.CreatePool(poolName)
			if err != nil {
				logrus.Errorf("f5 AddLBConfig: Error creating pool: %s , err: %v\n", poolName, err)
				return err
			}
		}

		var pool *bigip.Pool
		pool, err = client.GetPool(poolName)
		if err != nil {
			logrus.Errorf("f5 AddLBConfig: Error getting back the pool: %v\n", err)
			return err
		} else {
			pool.AllowNAT = true
			pool.AllowSNAT = true
			err = client.ModifyPool(poolName, pool)
			if err != nil {
				logrus.Errorf("f5 AddLBConfig: Error modifying the pool: %v\n", err)
				return err
			}
		}

		// Add members to our pool if not already present

		poolMembers, err := client.PoolMembers(poolName)
		if err != nil {
			logrus.Errorf("f5 AddLBConfig: Error listing members of  pool: %v\n", err)
			return err
		}
		for _, node := range nodes {
			if !poolMemberExists(poolMembers, node.HostIP+":"+node.Port) {
				err = client.AddPoolMember(poolName, node.HostIP+":"+node.Port)
				if err != nil {
					logrus.Errorf("f5 AddLBConfig: Error adding member to pool: %v\n", err)
					return err
				}
			}
		}

		//Add pool to virtualserver provided
		updatedVs := bigip.VirtualServer{}
		updatedVs.Pool = poolName

		err = client.ModifyVirtualServer(config.LBEndpoint, &updatedVs)

		if err != nil {
			logrus.Errorf("f5 AddLBConfig: Error modifying virtual server: %v\n", err)
			return err
		} else {
			logrus.Debugf("f5 AddLBConfig: Success adding the LB config to f5")
		}

	}
	return nil
}

func nodeExists(name string, nodeIp string) bool {
	bigIpNode, err := client.GetNode(name)
	if err != nil {
		logrus.Errorf("f5: Error getting f5 node: %v\n", err)
		return false
	}
	if bigIpNode != nil && bigIpNode.Address == nodeIp {
		return true
	}

	return false
}

func poolExists(name string) bool {
	bigIpPool, err := client.GetPool(name)
	if err != nil {
		logrus.Errorf("f5: Error getting f5 pool: %v\n", err)
		return false
	}

	if bigIpPool != nil {
		return true
	}

	return false
}

func poolMemberExists(poolMembers []string, member string) bool {
	for _, a := range poolMembers {
		if a == member {
			return true
		}
	}
	return false
}

//delete the LBConfig (unassign pool from virtualServer, remove pool, remove nodes)
func (*F5BigIPHandler) RemoveLBConfig(config model.LBConfig) error {
	_, err := client.GetVirtualServer(config.LBEndpoint)
	if err != nil {
		logrus.Errorf("f5 RemoveLBConfig: Error getting f5 virtual server: %v\n", err)
		return err
	}
	//virtualserver exists,
	//Remove pool from virtualserver provided
	updatedVs := bigip.VirtualServer{}
	updatedVs.Pool = "None"

	err = client.ModifyVirtualServer(config.LBEndpoint, &updatedVs)

	if err != nil {
		logrus.Errorf("f5 RemoveLBConfig: Error modifying virtual server: %v\n", err)
		return err
	}

	poolName := config.LBTargetPoolName

	poolMembers, err := client.PoolMembers(poolName)
	var nodes []model.LBTarget
	if err != nil {
		logrus.Errorf("f5 RemoveLBConfig: Error listing pool members for pool: %s, err: %v\n", poolName, err)
	} else {
		for _, member := range poolMembers {
			nodeParts := strings.Split(member, ":")
			if len(nodeParts) == 2 {
				node := model.LBTarget{}
				node.HostIP = nodeParts[0]
				node.Port = nodeParts[1]
				nodes = append(nodes, node)
			}
		}
	}
	//remove the pool
	err = client.DeletePool(poolName)
	if err != nil {
		logrus.Errorf("f5 RemoveLBConfig: Error removing pool: %s , err: %v\n", poolName, err)
	}
	//remove the nodes under the pool
	for _, node := range nodes {
		if nodeExists(node.HostIP, node.HostIP) {
			//node exist, delete node
			err = client.DeleteNode(node.HostIP)
			if err != nil {
				logrus.Errorf("f5 RemoveLBConfig: Error removing node on f5: %v\n", err)
			}
		}
	}

	logrus.Debugf("f5 RemoveLBConfig: Success")
	return nil
}

func (f *F5BigIPHandler) UpdateLBConfig(config model.LBConfig) error {
	err := f.RemoveLBConfig(config)
	if err != nil {
		logrus.Errorf("f5 UpdateLBConfig: Error removing existing config: %v\n", err)
		return err
	}

	err = f.AddLBConfig(config)
	if err != nil {
		logrus.Errorf("f5 UpdateLBConfig: Error adding back the config: %v\n", err)
		return err
	}

	logrus.Debugf("f5 UpdateLBConfig: Success")
	return nil
}

func (*F5BigIPHandler) GetLBConfigs() ([]model.LBConfig, error) {
	//list all virtualServers
	// for each vs -> LBEndpoint
	// get the pool -> LBTargetPoolName
	// pool members -> LB Targets hostIP : Port
	var lbConfigs []model.LBConfig

	vServers, err := client.VirtualServers()
	if err != nil {
		logrus.Errorf("f5 GetLBConfigs: Error listing f5 virtual servers: %v\n", err)
		return lbConfigs, err
	}

	for _, vServer := range vServers.VirtualServers {
		if vServer.Pool != "" {
			pool, err := client.GetPool(strings.TrimPrefix(vServer.Pool, "/Common/"))
			if err != nil {
				logrus.Errorf("f5 GetLBConfigs: Error getting the pool: %s, err: %v\n", vServer.Pool, err)
				continue
			}
			lbConfig := model.LBConfig{}
			lbConfig.LBEndpoint = vServer.Name
			lbConfig.LBTargetPoolName = pool.Name

			var nodes []model.LBTarget

			poolMembers, err := client.PoolMembers(pool.Name)
			if err != nil {
				logrus.Errorf("f5 GetLBConfigs: Error listing pool members for pool: %s, err: %v\n", pool.Name, err)
			} else {
				for _, member := range poolMembers {
					nodeParts := strings.Split(member, ":")
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

func (*F5BigIPHandler) TestConnection() error {
	return checkF5Connection()
}


func checkF5Connection() error {
	_, err := client.Pools()
	if err != nil {
		logrus.Errorf("f5 TestConnection: Error listing f5 pool: %v\n", err)
	} else {
		logrus.Infof("f5 TestConnection check passed")
	}
	return err
}