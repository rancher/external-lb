package avi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

const (
	ProviderName = "Avi"

	AVI_USER                 = "AVI_USER"
	AVI_PASSWORD             = "AVI_PASSWORD"
	AVI_CONTROLLER_ADDR      = "AVI_CONTROLLER_ADDR"
	AVI_CONTROLLER_PORT      = "AVI_CONTROLLER_PORT"
	AVI_SSL_VERIFY           = "AVI_SSL_VERIFY"
	AVI_CA_CERT_PATH         = "AVI_CA_CERT_PATH"
	AVI_CLOUD_NAME           = "AVI_CLOUD_NAME"
	LB_TARGET_RANCHER_SUFFIX = "LB_TARGET_RANCHER_SUFFIX"

	// Assume VSes already configured, so following not needed
	// AVI_DNS_SUBDOMAIN   = "AVI_DNS_SUBDOMAIN"

	// Avi password configured as avi-creds secret in Rancher
	AVI_SECRETES_FILE = "/run/secrets/avi-creds"
)

type AviConfig struct {
	controllerIpAddr string
	controllerPort   int
	username         string
	password         string
	sslVerify        bool
	caCertPath       string
	cloudName        string
	dnsSubDomain     string
	lbSuffix         string
}

func getAviPasswd() string {
	// check is passwordis available via Rancher secrets, if not, then
	// look for it in environment variable
	data, err := ioutil.ReadFile(AVI_SECRETES_FILE)
	if err != nil {
		log.Warnf("Error reading secrets file: %s", err)
	} else {
		return string(data)
	}

	return os.Getenv(AVI_PASSWORD)
}

func GetAviConfig() (*AviConfig, error) {
	conf := make(map[string]string)
	conf[AVI_USER] = os.Getenv(AVI_USER)
	conf[AVI_CONTROLLER_ADDR] = os.Getenv(AVI_CONTROLLER_ADDR)
	conf[AVI_CONTROLLER_PORT] = os.Getenv(AVI_CONTROLLER_PORT)
	conf[AVI_SSL_VERIFY] = os.Getenv(AVI_SSL_VERIFY)
	conf[AVI_CA_CERT_PATH] = os.Getenv(AVI_CA_CERT_PATH)
	conf[LB_TARGET_RANCHER_SUFFIX] = os.Getenv(LB_TARGET_RANCHER_SUFFIX)

	// Assume VSes already configured, so following not needed
	// conf[AVI_CLOUD_NAME] = os.Getenv(AVI_CLOUD_NAME)
	// conf[AVI_DNS_SUBDOMAIN] = os.Getenv(AVI_DNS_SUBDOMAIN)

	conf[AVI_PASSWORD] = getAviPasswd()

	b, _ := json.MarshalIndent(conf, "", " ")
	log.Infof("Configured provider %s with values %s \n",
		ProviderName, string(b))

	return validateConfig(conf)
}

func validateConfig(conf map[string]string) (*AviConfig, error) {
	cfg := new(AviConfig)
	if conf[AVI_USER] == "" {
		return cfg, fmt.Errorf("AVI_USER not set")
	}
	cfg.username = conf[AVI_USER]

	if conf[AVI_PASSWORD] == "" {
		return cfg, fmt.Errorf("AVI_PASSWORD not set")
	}
	cfg.password = conf[AVI_PASSWORD]

	if conf[AVI_CONTROLLER_ADDR] == "" {
		return cfg, fmt.Errorf("AVI_CONTROLLER_ADDR not set")
	}
	cfg.controllerIpAddr = conf[AVI_CONTROLLER_ADDR]

	var port int
	var err error
	if conf[AVI_CONTROLLER_PORT] == "" {
		log.Warn("AVI_CONTROLLER_PORT is not set, using 443 as default.")
		port = 443
	} else {
		port, err = strconv.Atoi(conf[AVI_CONTROLLER_PORT])
		if err != nil {
			log.Errorf("Error controller port: %v", err)
			return cfg, err
		}
	}
	cfg.controllerPort = port

	sslVerify := true
	if conf[AVI_SSL_VERIFY] == "no" || conf[AVI_SSL_VERIFY] == "false" {
		conf[AVI_SSL_VERIFY] = "false"
		sslVerify = false
	} else {
		conf[AVI_SSL_VERIFY] = "true"
	}

	if sslVerify {
		if conf[AVI_CA_CERT_PATH] != "" {
			// check if path exists
			log.Info("Using system default path for CA certificates")
		} else {
			log.Infof("Using CA certificate path %s", conf[AVI_CA_CERT_PATH])
		}
	}
	cfg.sslVerify = sslVerify
	cfg.caCertPath = conf[AVI_CA_CERT_PATH]

	if conf[AVI_CLOUD_NAME] == "" {
		log.Info("AVI_CLOUD_NAME not set, using Default-Cloud")
		conf[AVI_CLOUD_NAME] = "Default-Cloud"
	}
	cfg.cloudName = conf[AVI_CLOUD_NAME]

	//if conf[AVI_DNS_SUBDOMAIN] == "" {
	//log.Info("AVI_DNS_SUBDOMAIN not set")
	//}
	cfg.dnsSubDomain = ""

	if conf[LB_TARGET_RANCHER_SUFFIX] == "" {
		conf[LB_TARGET_RANCHER_SUFFIX] = "rancher.internal"
	}
	cfg.lbSuffix = conf[LB_TARGET_RANCHER_SUFFIX]

	return cfg, nil
}
