package zevenetlb

import (
	"fmt"
	"strconv"
	"strings"
)

// OptionalBool represents an optional boolean having an *undefined/nil* state.
type OptionalBool string

const (
	OptionalBool_Nil   OptionalBool = ""
	OptionalBool_True  OptionalBool = "true"
	OptionalBool_False OptionalBool = "false"
)

type genericResponse struct {
	Description string `json:"description"`
}

// FarmProfile is an enumeration of possible farm profiles.
type FarmProfile string

const (
	FarmProfile_HTTP      FarmProfile = "http"
	FarmProfile_HTTPS     FarmProfile = "https"
	FarmProfile_Level4NAT FarmProfile = "l4xnat"
	FarmProfile_DataLink  FarmProfile = "datalink"
)

type farmListResponse struct {
	Description string     `json:"description"`
	Params      []FarmInfo `json:"params"`
}

// FarmInfo contains the list of all available farms.
// See https://www.zevenet.com/zapidoc_ce_v3.1/#list-all-farms
type FarmInfo struct {
	FarmName    string      `json:"farmname"`
	Profile     FarmProfile `json:"profile"`
	Status      FarmStatus  `json:"status"`
	VirtualIP   string      `json:"vip"`
	VirtualPort int         `json:"vport,string"`
}

// String returns the farm's name and profile.
func (fi *FarmInfo) String() string {
	return fmt.Sprintf("%v (%v)", fi.FarmName, fi.Profile)
}

// GetAllFarms returns list os all available farms.
func (s *ZapiSession) GetAllFarms() ([]FarmInfo, error) {
	var result *farmListResponse

	err := s.getForEntity(&result, "farms")

	if err != nil {
		return nil, err
	}

	return result.Params, nil
}

type farmDetailsResponse struct {
	Description string           `json:"description"`
	Params      FarmDetails      `json:"params"`
	Services    []ServiceDetails `json:"services"`
}

// FarmCiphers is an enumeration of possible selections of *Ciphers* to be used for an https listener.
// The custom cipher requires *CiphersCustom* to bet set to an OpenSSL compatible ciphers string.
type FarmCiphers string

const (
	FarmCiphers_All    FarmCiphers = "all"
	FarmCiphers_High   FarmCiphers = "highsecurity"
	FarmCiphers_Custom FarmCiphers = "customsecurity"
)

// FarmHTTPVerb is an enumeration of possible *HTTPVerbs* to be used for an http/s listener.
type FarmHTTPVerb string

const (
	// FarmHTTPVerb_Standard accepts GET, POST, HEAD
	FarmHTTPVerb_Standard FarmHTTPVerb = "standardHTTP"

	// FarmHTTPVerb_Extended accepts GET, POST, HEAD, PUT, DELETE
	FarmHTTPVerb_Extended FarmHTTPVerb = "extendedHTTP"

	// FarmHTTPVerb_WebDAV accepts GET, POST, HEAD, PUT, DELETE, LOCK, UNLOCK, PROPFIND, PROPPATCH, SEARCH, MKCOL, MOVE, COPY, OPTIONS, TRACE, MKACTIVITY, CHECKOUT, MERGE, REPORT
	FarmHTTPVerb_WebDAV FarmHTTPVerb = "standardWebDAV"

	// FarmHTTPVerb_MicrosoftWebDAV accepts GET, POST, HEAD, PUT, DELETE, LOCK, UNLOCK, PROPFIND, PROPPATCH, SEARCH, MKCOL, MOVE, COPY, OPTIONS, TRACE, MKACTIVITY, CHECKOUT, MERGE, REPORT, SUBSCRIBE, UNSUBSCRIBE, NOTIFY, BPROPFIND, BPROPPATCH, POLL, BMOVE, BCOPY, BDELETE, CONNECT
	FarmHTTPVerb_MicrosoftWebDAV FarmHTTPVerb = "MSextWebDAV"

	// FarmHTTPVerb_MicrosoftRPC accepts GET, POST, HEAD, PUT, DELETE, LOCK, UNLOCK, PROPFIND, PROPPATCH, SEARCH, MKCOL, MOVE, COPY, OPTIONS, TRACE, MKACTIVITY, CHECKOUT, MERGE, REPORT, SUBSCRIBE, UNSUBSCRIBE, NOTIFY, BPROPFIND, BPROPPATCH, POLL, BMOVE, BCOPY, BDELETE, CONNECT, RPC_IN_DATA, RPC_OUT_DATA
	FarmHTTPVerb_MicrosoftRPC FarmHTTPVerb = "MSRPCext"
)

// FarmListener is an enumeration of possible selections of *Listener* values.
type FarmListener string

const (
	FarmListener_HTTP  FarmListener = "http"
	FarmListener_HTTPS FarmListener = "https"
)

// FarmRewriteLocation is an enumeration of possible selections of *RewriteLocation* values.
// If it is enabled, the farm is forced to modify the Location: and Content-location: headers in responses to clients with the virtual host.
type FarmRewriteLocation string

const (
	FarmRewriteLocation_Enabled      FarmRewriteLocation = "enabled"
	FarmRewriteLocation_BackendsOnly FarmRewriteLocation = "enabled-backends"
	FarmRewriteLocation_Disabled     FarmRewriteLocation = "disabled"
)

// FarmStatus is an enumeration of possible selections of *Status* values.
type FarmStatus string

const (
	// FarmStatus_Up means the farm is up and all the backends are working fine.
	FarmStatus_Up FarmStatus = "up"

	// FarmStatus_Down means the farm is not running. Use *StartFarm()* to start the farm.
	FarmStatus_Down FarmStatus = "down"

	// FarmStatus_NeedsRestart means the farm is up but it is pending of a restart action. Use *RestartFarm()* to perform the required restart.
	FarmStatus_NeedsRestart FarmStatus = "needed restart"

	// FarmStatus_Critical means the farm is up and all backends are unreachable or maintenance. This is usually the case for newly created empty farms.
	FarmStatus_Critical FarmStatus = "critical"

	// FarmStatus_Problem means the farm is up and there are some backend unreachable, but at least one backend is in up status.
	FarmStatus_Problem FarmStatus = "problem"

	// FarmStatus_Maintenance means the farm is up and there are backends in up status, but at least one backend is in maintenance mode. Use *SetServiceMaintenance(false)* on the service.
	FarmStatus_Maintenance FarmStatus = "maintenance"
)

// FarmDetails contains all information regarding a farm and the services.
// See https://www.zevenet.com/zapidoc_ce_v3.1/#retrieve-farm-by-name
type FarmDetails struct {
	Certificates             []CertificateInfo   `json:"certlist"`
	FarmName                 string              `json:"farmname"`
	CiphersCustom            string              `json:"cipherc,omitempty"`
	Ciphers                  FarmCiphers         `json:"ciphers,omitempty"`
	ConnectionTimeoutSeconds int                 `json:"contimeout"`
	DisableSSLv2             bool                `json:"disable_sslv2,string"`
	DisableSSLv3             bool                `json:"disable_sslv3,string"`
	DisableTLSv1             bool                `json:"disable_tlsv1,string"`
	DisableTLSv11            bool                `json:"disable_tlsv1_1,string"`
	DisableTLSv12            bool                `json:"disable_tlsv1_2,string"`
	ErrorString414           string              `json:"error414"`
	ErrorString500           string              `json:"error500"`
	ErrorString501           string              `json:"error501"`
	ErrorString503           string              `json:"error503"`
	HTTPVerbs                FarmHTTPVerb        `json:"httpverb"`
	Listener                 FarmListener        `json:"listener"`
	RequestTimeoutSeconds    int                 `json:"reqtimeout"`
	ResponseTimeoutSeconds   int                 `json:"restimeout"`
	ResurrectIntervalSeconds int                 `json:"resurrectime"`
	RewriteLocation          FarmRewriteLocation `json:"rewritelocation"`
	Status                   FarmStatus          `json:"status"`
	VirtualIP                string              `json:"vip"`
	VirtualPort              int                 `json:"vport"`
	Services                 []ServiceDetails    `json:"services"`
}

// String returns the farm's name and listener.
func (fd *FarmDetails) String() string {
	return fmt.Sprintf("%v (%v)", fd.FarmName, fd.Listener)
}

// IsHTTP checks if the farm has HTTP or HTTPS support enabled.
func (fd *FarmDetails) IsHTTP() bool {
	return strings.HasPrefix(string(fd.Listener), "http")
}

// IsRunning checks if the farm is up and running.
func (fd *FarmDetails) IsRunning() bool {
	return fd.Status == FarmStatus_Up
}

// GetService retrieves a service by its name, or returns *nil* if not found.
func (fd *FarmDetails) GetService(serviceName string) (*ServiceDetails, error) {
	for _, s := range fd.Services {
		if s.ServiceName == serviceName {
			return &s, nil
		}
	}

	return nil, nil
}

// GetFarm returns details on a specific farm.
func (s *ZapiSession) GetFarm(farmName string) (*FarmDetails, error) {
	var result *farmDetailsResponse

	err := s.getForEntity(&result, "farms", farmName)

	if err != nil {
		// farm not found?
		if strings.Contains(err.Error(), "Farm not found") {
			return nil, nil
		}

		return nil, err
	}

	// inject values
	if result != nil {
		result.Params.FarmName = farmName
		result.Params.Services = result.Services

		for s := range result.Params.Services {
			service := &result.Params.Services[s]

			service.FarmName = farmName

			for b := range service.Backends {
				backend := &service.Backends[b]

				backend.FarmName = farmName
				backend.ServiceName = service.ServiceName
			}
		}
	}

	return &result.Params, nil
}

// DeleteFarm will delete an existing farm (or do nothing if missing)
func (s *ZapiSession) DeleteFarm(farmName string) (bool, error) {
	// retrieve farm details
	farm, err := s.GetFarm(farmName)

	if err != nil {
		return false, err
	}

	// farm does not exist?
	if farm == nil {
		return false, nil
	}

	// delete the farm
	return true, s.delete("farms", farmName)
}

type farmCreate struct {
	FarmName    string `json:"farmname"`
	Profile     string `json:"profile"`
	VirtualIP   string `json:"vip"`
	VirtualPort int    `json:"vport"`
}

// CreateFarmAsHTTP creates a new HTTP farm.
// A newly created farm is in the *critical* state, due to the lack of services and backends.
// The *virtualPort* is optional and can be 0, using port 80 as default.
func (s *ZapiSession) CreateFarmAsHTTP(farmName string, virtualIP string, virtualPort int) (*FarmDetails, error) {
	// set default HTTP port
	if virtualPort <= 0 {
		virtualPort = 80
	}

	// create the farm
	req := farmCreate{
		FarmName:    farmName,
		Profile:     "http",
		VirtualIP:   virtualIP,
		VirtualPort: virtualPort,
	}

	err := s.post(req, "farms")

	if err != nil {
		return nil, err
	}

	// retrieve status
	return s.GetFarm(farmName)
}

// CreateFarmAsHTTPS creates a new HTTPS farm.
// A newly created farm is in the *critical* state, due to the lack of services and backends.
// The *virtualPort* is optional and can be 0, using port 443 as default.
func (s *ZapiSession) CreateFarmAsHTTPS(farmName string, virtualIP string, virtualPort int, certFilename string) (*FarmDetails, error) {
	// set default HTTPS port
	if virtualPort <= 0 {
		virtualPort = 443
	}

	// create the farm
	farm, err := s.CreateFarmAsHTTP(farmName, virtualIP, virtualPort)

	if err != nil {
		return nil, err
	}

	// update the farm
	farm.Listener = "https"
	farm.Ciphers = "highsecurity"
	farm.DisableSSLv2 = false
	farm.DisableSSLv3 = false
	farm.DisableTLSv1 = false

	s.UpdateFarm(farm)

	return farm, nil
}

// UpdateFarm updates the HTTP/S farm.
// This method does *not* update the *services*. Use *UpdateService()* instead.
func (s *ZapiSession) UpdateFarm(farm *FarmDetails) error {
	return s.put(farm, "farms", farm.FarmName)
}

type farmAction struct {
	Action string `json:"action"`
}

// StartFarm will start a stopped farm.
func (s *ZapiSession) StartFarm(farmName string) error {
	req := farmAction{Action: "start"}

	return s.put(req, "farms", farmName, "actions")
}

// StopFarm will stop a running farm.
func (s *ZapiSession) StopFarm(farmName string) error {
	req := farmAction{Action: "stop"}

	return s.put(req, "farms", farmName, "actions")
}

// RestartFarm will restart a running farm.
func (s *ZapiSession) RestartFarm(farmName string) error {
	req := farmAction{Action: "restart"}

	return s.put(req, "farms", farmName, "actions")
}

// CertificateInfo contains reference information on a certificate.
type CertificateInfo struct {
	Filename string `json:"file"`
	ID       int    `json:"id"`
}

// String returns the certificate's filename.
func (ci CertificateInfo) String() string {
	return ci.Filename
}

type serviceDetailsResponse struct {
	Description string         `json:"description"`
	Params      ServiceDetails `json:"params"`
}

// ServiceRedirectType is an enumeration of possible selections of *RedirectType* values.
type ServiceRedirectType string

const (
	// ServiceRedirectType_Default means the url is taken as an absolute host and path to redirect to.
	ServiceRedirectType_Default ServiceRedirectType = "default"

	// ServiceRedirectType_Append means the original request path or URI will be appended to the host and path.
	ServiceRedirectType_Append ServiceRedirectType = "append"

	// ServiceRedirectType_Disabled means the *RedirectURL* field is not set.
	ServiceRedirectType_Disabled ServiceRedirectType = ""
)

// ServiceConnPersistenceMode is an enumeration of possible selections of *ConnectionPersistenceMode* values.
type ServiceConnPersistenceMode string

const (
	// ServiceConnPersistenceMode_Disabled means no action is taken.
	ServiceConnPersistenceMode_Disabled ServiceConnPersistenceMode = ""

	// ServiceConnPersistenceMode_IPAddress means the persistence session is done in base of client IP.
	ServiceConnPersistenceMode_IPAddress ServiceConnPersistenceMode = "IP"

	// ServiceConnPersistenceMode_BasicHeaders means the persistence session is done in base of BASIC headers.
	ServiceConnPersistenceMode_BasicHeaders ServiceConnPersistenceMode = "BASIC"

	// ServiceConnPersistenceMode_Url means the persistence session is done in base of a field in the URI. Set the query parameter name in *ConnectionPersistenceID*.
	ServiceConnPersistenceMode_Url ServiceConnPersistenceMode = "URL"

	// ServiceConnPersistenceMode_QueryParameter means the persistence session is done in base of a value at the end of the URI.
	ServiceConnPersistenceMode_QueryParameter ServiceConnPersistenceMode = "PARM"

	// ServiceConnPersistenceMode_Cookie means the persistence session is done in base of a cookie name, this cookie has to be created by the backends! Set the cookie name in *ConnectionPersistenceID*.
	ServiceConnPersistenceMode_Cookie ServiceConnPersistenceMode = "COOKIE"

	// ServiceConnPersistenceMode_Header means the persistence session is done in base of a Header name. Set the header name in *ConnectionPersistenceID*.
	ServiceConnPersistenceMode_Header ServiceConnPersistenceMode = "HEADER"
)

// ServiceDetails contains all information regarding a single service.
type ServiceDetails struct {
	ServiceName                         string                     `json:"id"`
	FarmGuardianEnabled                 bool                       `json:"fgenabled,string"`
	FarmGuardianLogsEnabled             OptionalBool               `json:"fglog"`
	FarmGuardianScript                  string                     `json:"fgscript"`
	FarmGuardianCheckIntervalSeconds    int                        `json:"fgtimecheck"`
	EncryptedBackends                   bool                       `json:"httpsb,string"`
	LastResponseBalancingEnabled        bool                       `json:"leastresp,string"`
	ConnectionPersistenceMode           ServiceConnPersistenceMode `json:"persistence"`
	ConnectionPersistenceID             string                     `json:"sessionid"`
	ConnectionPersistenceTimeoutSeconds int                        `json:"ttl"`
	RedirectURL                         string                     `json:"redirect"`
	RedirectType                        ServiceRedirectType        `json:"redirecttype"`
	URLPattern                          string                     `json:"urlp"`
	HostPattern                         string                     `json:"vhost"`
	Backends                            []BackendDetails           `json:"backends"`
	FarmName                            string                     `json:"farmname"`
}

// String returns the services' name.
func (sd ServiceDetails) String() string {
	return sd.ServiceName
}

// GetBackend retrieves a backend by its ID, or returns *nil* if not found.
func (sd *ServiceDetails) GetBackend(backendID int) (*BackendDetails, error) {
	for _, s := range sd.Backends {
		if s.ID == backendID {
			return &s, nil
		}
	}

	return nil, nil
}

// GetBackendByAddress retrieves a backend by its IP address and port, or returns *nil* if not found. The *port* is optional and can be 0.
func (sd *ServiceDetails) GetBackendByAddress(ipAddress string, port int) (*BackendDetails, error) {
	for _, s := range sd.Backends {
		if s.IPAddress == ipAddress && (port <= 0 || s.Port == port) {
			return &s, nil
		}
	}

	return nil, nil
}

type serviceCreate struct {
	ServiceName string `json:"id"`
}

// DeleteService will delete an existing service (or do nothing if service or farm is missing)
func (s *ZapiSession) DeleteService(farmName string, serviceName string) (bool, error) {
	// retrieve farm details
	farm, err := s.GetFarm(farmName)

	if err != nil {
		return false, err
	}

	// farm does not exist?
	if farm == nil {
		return false, nil
	}

	// does the service exist?
	service, err := farm.GetService(serviceName)

	if err != nil {
		return false, err
	}

	if service == nil {
		return false, nil
	}

	// delete the service
	return true, s.delete("farms", farmName, "services", serviceName)
}

// CreateService creates a new service on a farm.
func (s *ZapiSession) CreateService(farmName string, serviceName string) (*ServiceDetails, error) {
	// create the service
	req := serviceCreate{
		ServiceName: serviceName,
	}

	err := s.post(req, "farms", farmName, "services")

	if err != nil {
		return nil, err
	}

	// retrieve status
	farm, err := s.GetFarm(farmName)

	if err != nil {
		return nil, err
	}

	return farm.GetService(serviceName)
}

type farmguardianUpdate struct {
	ServiceName                      string       `json:"service"`
	FarmGuardianEnabled              bool         `json:"fgenabled,string"`
	FarmGuardianLogsEnabled          OptionalBool `json:"fglog"`
	FarmGuardianScript               string       `json:"fgscript"`
	FarmGuardianCheckIntervalSeconds int          `json:"fgtimecheck"`
}

// UpdateService updates a service on a farm.
// This method does *not* update the *backends*. Use *UpdateBackend()* instead.
func (s *ZapiSession) UpdateService(service *ServiceDetails) error {
	err := s.put(service, "farms", service.FarmName, "services", service.ServiceName)

	if err != nil {
		return err
	}

	// update farm guardian
	fg := farmguardianUpdate{
		ServiceName:                      service.ServiceName,
		FarmGuardianEnabled:              service.FarmGuardianEnabled,
		FarmGuardianScript:               service.FarmGuardianScript,
		FarmGuardianCheckIntervalSeconds: service.FarmGuardianCheckIntervalSeconds,
		FarmGuardianLogsEnabled:          service.FarmGuardianLogsEnabled,
	}

	if fg.FarmGuardianScript == "" {
		fg.FarmGuardianScript = "check_http -H HOST -p PORT"
	}

	return s.put(fg, "farms", service.FarmName, "fg")
}

type backendDetailsResponse struct {
	Description string         `json:"description"`
	Params      BackendDetails `json:"params"`
}

// BackendStatus is an enumeration of possible selections of *Status* values.
type BackendStatus string

const (
	// BackendStatus_Up means the backend is ready to receive connections.
	BackendStatus_Up BackendStatus = "up"

	// BackendStatus_Down means the backend is not working.
	BackendStatus_Down BackendStatus = "down"

	// BackendStatus_Maintenance means backend is marked as not ready for receiving connections by the administrator. Use *SetServiceMaintenance(false)* on the service.
	BackendStatus_Maintenance BackendStatus = "maintenance"

	// BackendStatus_Undefined means the backend status has been not checked.
	BackendStatus_Undefined BackendStatus = "undefined"
)

// BackendDetails contains all information regarding a single backend server.
type BackendDetails struct {
	ID             int           `json:"id"`
	IPAddress      string        `json:"ip"`
	Port           int           `json:"port"`
	Status         BackendStatus `json:"status"`
	TimeoutSeconds *int          `json:"timeout,omitempty"`
	Weight         *int          `json:"weight,omitempty"`
	FarmName       string        `json:"farmname"`
	ServiceName    string        `json:"servicename"`
}

// String returns the backend's IP, port, ID, and status.
func (bd BackendDetails) String() string {
	return fmt.Sprintf("%v:%v (ID: %v, Status: %v)", bd.IPAddress, bd.Port, bd.ID, bd.Status)
}

type backendCreate struct {
	IPAddress string `json:"ip"`
	Port      int    `json:"port"`
}

// DeleteBackend will delete an existing backend (or do nothing if backend or service or farm is missing)
func (s *ZapiSession) DeleteBackend(farmName string, serviceName string, backendId int) (bool, error) {
	// retrieve farm details
	farm, err := s.GetFarm(farmName)

	if err != nil {
		return false, err
	}

	// farm does not exist?
	if farm == nil {
		return false, nil
	}

	// does the service exist?
	service, err := farm.GetService(serviceName)

	if err != nil {
		return false, err
	}

	if service == nil {
		return false, nil
	}

	// does the backend exist?
	backend, err := service.GetBackend(backendId)

	if err != nil {
		return false, err
	}

	if backend == nil {
		return false, nil
	}

	// delete the backend
	return true, s.delete("farms", farmName, "services", serviceName, "backends", strconv.Itoa(backendId))
}

// CreateBackend creates a new backend on a service on a farm.
func (s *ZapiSession) CreateBackend(farmName string, serviceName string, backendIP string, backendPort int) (*BackendDetails, error) {
	// create the backend
	req := backendCreate{
		IPAddress: backendIP,
		Port:      backendPort,
	}

	err := s.post(req, "farms", farmName, "services", serviceName, "backends")

	if err != nil {
		return nil, err
	}

	// retrieve status
	farm, err := s.GetFarm(farmName)

	if err != nil {
		return nil, err
	}

	service, err := farm.GetService(serviceName)

	if err != nil {
		return nil, err
	}

	return service.GetBackendByAddress(backendIP, backendPort)
}

// UpdateBackend updates a backend on a service on a farm.
func (s *ZapiSession) UpdateBackend(backend *BackendDetails) error {
	return s.put(backend, "farms", backend.FarmName, "services", backend.ServiceName, "backends", strconv.Itoa(backend.ID))
}

type backendMaintenance struct {
	Action string `json:"action"`
	Mode   string `json:"mode,omitempty"`
}

// SetBackendMaintenance updates a backend on a service on a farm.
// To cut and disconnect any existing connections when enabling maintenance, set *cutExistingConnections* to true.
func (s *ZapiSession) SetBackendMaintenance(backend *BackendDetails, enableMaintenance bool, cutExistingConnections bool) error {
	var cmd backendMaintenance

	if enableMaintenance {
		var mode string

		if cutExistingConnections {
			mode = "cut"
		} else {
			mode = "drain"
		}

		cmd = backendMaintenance{
			Action: "maintenance",
			Mode:   mode,
		}
	} else {
		// recover from maintenance
		cmd = backendMaintenance{
			Action: "up",
		}
	}

	return s.put(cmd, "farms", backend.FarmName, "services", backend.ServiceName, "backends", strconv.Itoa(backend.ID), "maintenance")
}
