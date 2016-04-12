package f5

import (  
    //"fmt"
    "github.com/scottdware/go-bigip"
    "github.com/Sirupsen/logrus"
    "github.com/rancher/external-lb/providers"
    "github.com/rancher/external-lb/model"
	"os"
)

const (
	name = "f5_BigIP"
)

var (
	client     *bigip.BigIP
)

func init() {
	f5_host := os.Getenv("F5_BIGIP_HOST")
	if len(f5_host) == 0 {
		logrus.Fatalf("F5_BIGIP_HOST is not set, skipping init of %s provider", name)
		return
	}
	logrus.Infof("%s",f5_host)
	
	f5_admin := os.Getenv("F5_BIGIP_USER")
	if len(f5_admin) == 0 {
		logrus.Fatalf("F5_BIGIP_USER is not set, skipping init of %s provider", name)
		return
	}
	logrus.Infof("%s", f5_admin)

	f5_pwd := os.Getenv("F5_BIGIP_PWD")
	if len(f5_pwd) == 0 {
		logrus.Fatalf("F5_BIGIP_PWD is not set, skipping init of %s provider", name)
		return
	}
	logrus.Infof("%s", f5_pwd)

	client = bigip.NewSession(f5_host, f5_admin, f5_pwd)

	f5BigIPHandler := &F5BigIPHandler{}
	if err := providers.RegisterProvider(name, f5BigIPHandler); err != nil {
		logrus.Fatalf("Could not register %s provider",name)
	}
	
	logrus.Infof("Configured %s LB provider", f5BigIPHandler.GetName())

}


type F5BigIPHandler struct {
}

func (*F5BigIPHandler) GetName() string {
	return name
}

func (*F5BigIPHandler) ConfigureLBRecord(rec model.LBRecord) error {
	
	_, err := client.GetVirtualServer(rec.Vip)
    if err != nil {
   	 	logrus.Errorf("Error getting f5 virtual server: %v\n", err)
   	 	return err
	} else {
		//virtualserver exists, add nodes and pool
		
		nodes := rec.Nodes
		for _, node := range nodes {
			err = client.CreateNode(node.HostIP, node.HostIP)
		    if err != nil {
		   	 	logrus.Errorf("Error creating node on f5: %v\n", err)
		   	 	return err
			} else {
				logrus.Debugf("Success creating node %s", node.HostIP)
			}
		}
		
		//Create our pool
		poolName := rec.Vip + "_" + rec.ServiceName
	    err = client.CreatePool(poolName)
	    if err != nil {
   	 		logrus.Errorf("Error creating pool: %s , err: %v\n", poolName, err)
   	 		return err
		} else {
			var pool *bigip.Pool
			pool, err = client.GetPool(poolName)
			if err != nil {
   	 			logrus.Errorf("Error getting back the pool: %v\n", err)
   	 			return err
			} else {
				pool.AllowNAT = true
				pool.AllowSNAT = true
				err = client.ModifyPool(poolName, pool)
				if err != nil {
   	 				logrus.Errorf("Error modifying the pool: %v\n", err)
   	 				return err
				} 
			}
		}
		
		// Add members to our pool
		for _, node := range nodes {
			err = client.AddPoolMember(poolName, node.HostIP + ":" + node.Port)
			if err != nil {
	   	 		logrus.Errorf("Error adding member to pool: %v\n", err)
	   	 		return err
			} 
		}
		
		//Add pool to vip
		updatedVs := bigip.VirtualServer{}
    	updatedVs.Pool = poolName
    
   		err = client.ModifyVirtualServer(rec.Vip, &updatedVs)
    
	    if err != nil {
	   	 	logrus.Errorf("Error modifying virtual server: %v\n", err)
	   	 	return err
		} else {
			logrus.Debugf("Success adding the vip to f5")
		}

	}
	return nil
}