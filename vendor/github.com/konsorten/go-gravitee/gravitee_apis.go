package gravitee

import (
	"fmt"
	"strings"

	"github.com/Jeffail/gabs"
)

// ApiState is an enumeration of possible *State* to be used for an API.
type ApiState string

const (
	ApiState_Started ApiState = "started"
	ApiState_Stopped ApiState = "stopped"
)

// ApiVisibility is an enumeration of possible *Visibility* to be used for an API.
type ApiVisibility string

const (
	ApiVisibility_Private ApiVisibility = "private"
	ApiVisibility_Public  ApiVisibility = "public"
)

type ApiInfo struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Version         string        `json:"version"`
	Description     string        `json:"description"`
	Visibility      ApiVisibility `json:"visibility"`
	State           ApiState      `json:"state"`
	Views           []string      `json:"views"`
	Labels          []string      `json:"labels"`
	Manageable      bool          `json:"manageable"`
	NumberOfRatings int           `json:"numberOfRatings"`
	CreatedAt       int           `json:"created_at"`
	UpdatedAdd      int           `json:"updated_at"`
	Owner           UserReference `json:"owner"`
	PictureURL      string        `json:"picture_url"`
	ContextPath     string        `json:"context_path"`
}

func (ai ApiInfo) String() string {
	return fmt.Sprintf("%v (%v, %v)", ai.Name, ai.ID, ai.State)
}

// ApiMetadataFormat is an enumeration of possible *Format* to be used for an API metdata entry.
type ApiMetadataFormat string

const (
	ApiMetadataFormat_String  ApiMetadataFormat = "string"
	ApiMetadataFormat_Numeric ApiMetadataFormat = "numeric"
	ApiMetadataFormat_Boolean ApiMetadataFormat = "boolean"
	ApiMetadataFormat_Date    ApiMetadataFormat = "date"
	ApiMetadataFormat_Mail    ApiMetadataFormat = "mail"
	ApiMetadataFormat_URL     ApiMetadataFormat = "url"
)

type ApiMetadata struct {
	Key          string            `json:"key"`
	Name         string            `json:"name"`
	Format       ApiMetadataFormat `json:"format"`
	LocalValue   string            `json:"value"`
	DefaultValue string            `json:"defaultValue,omitempty"`
	ApiID        string            `json:"apiId,omitempty"`
}

func (ai ApiMetadata) Value() string {
	if ai.LocalValue != "" {
		return ai.LocalValue
	}

	return ai.DefaultValue
}

func (ai ApiMetadata) IsLocal() bool {
	return ai.ApiID != ""
}

func (ai ApiMetadata) String() string {
	return fmt.Sprintf("%v = %v [%v]", ai.Name, ai.Value(), ai.Format)
}

type ApiDetailsEndpointHttp struct {
	ConnectTimeoutMS         int  `json:"connectTimeout"`
	IdleTimeoutMS            int  `json:"idleTimeout"`
	ReadTimeoutMS            int  `json:"readTimeout"`
	KeepAlive                bool `json:"keepAlive"`
	Pipelining               bool `json:"pipelining"`
	MaxConcurrentConnections int  `json:"maxConcurrentConnections"`
	UseCompression           bool `json:"useCompression"`
	FollowRedirects          bool `json:"followRedirects"`
}

type ApiDetailsEndpointSSL struct {
	IsEnabled                  bool   `json:"enabled"`
	TrustAllCertificates       bool   `json:"trustAll"`
	VerifyHostnameInPublicCert bool   `json:"hostnameVerifier"`
	PublicCertPEM              string `json:"pem"`
}

type ApiDetailsEndpoint struct {
	Name     string                 `json:"name"`
	Target   string                 `json:"target"`
	Weight   int                    `json:"weight"`
	IsBackup bool                   `json:"backup"`
	Type     string                 `json:"type"`
	Http     ApiDetailsEndpointHttp `json:"http"`
	SSL      ApiDetailsEndpointSSL  `json:"ssl"`
}

func (ai ApiDetailsEndpoint) String() string {
	return fmt.Sprintf("%v (%v, %v)", ai.Name, ai.Target, ai.Type)
}

func MakeApiDetailsEndpoint(name, target string) ApiDetailsEndpoint {
	return ApiDetailsEndpoint{
		Name:   name,
		Target: target,
		Weight: 1,
		Type:   "HTTP",

		Http: ApiDetailsEndpointHttp{
			ConnectTimeoutMS:         5000,
			IdleTimeoutMS:            60000,
			ReadTimeoutMS:            10000,
			KeepAlive:                true,
			Pipelining:               false,
			MaxConcurrentConnections: 100,
			UseCompression:           true,
			FollowRedirects:          false,
		},

		SSL: ApiDetailsEndpointSSL{
			IsEnabled:                  false,
			TrustAllCertificates:       true,
			VerifyHostnameInPublicCert: false,
			PublicCertPEM:              "",
		},
	}
}

type ApiDetailsPath struct {
}

type ApiDetails struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Visibility  ApiVisibility `json:"visibility"`
	State       ApiState      `json:"state"`
	Tags        []string      `json:"tags"`
	Labels      []string      `json:"labels"`
	//Paths       map[string]ApiDetailsPath `json:"paths"`
	CreatedAt   int           `json:"created_at"`
	UpdatedAdd  int           `json:"updated_at"`
	DeployedAt  int           `json:"deployed_at"`
	Owner       UserReference `json:"owner"`
	PictureURL  string        `json:"picture_url"`
	ContextPath string        `json:"context_path"`
	Proxy       struct {
		ContextPath      string               `json:"context_path"`
		StripContextPath bool                 `json:"strip_context_path"`
		LoggingMode      string               `json:"loggingMode"`
		Endpoints        []ApiDetailsEndpoint `json:"endpoints"`

		LoadBalancing struct {
			Type string `json:"type"`
		} `json:"load_balancing"`

		CORS struct {
			IsEnabled        bool     `json:"enabled"`
			AllowCredentials bool     `json:"allowCredentials"`
			AllowHeaders     []string `json:"allowHeaders"`
			AllowMethods     []string `json:"allowMethods"`
			AllowOrigin      []string `json:"allowOrigin"`
			ExposeHeaders    []string `json:"exposeHeaders"`
			MaxAgeSeconds    int      `json:"maxAge"`
		} `json:"cors"`
	} `json:"proxy"`
}

func (ai ApiDetails) String() string {
	return fmt.Sprintf("%v (%v, %v)", ai.Name, ai.ID, ai.State)
}

// GetAllAPIs retrieves a list of all APIs registered in Gravitee.
func (s *GraviteeSession) GetAllAPIs() ([]ApiInfo, error) {
	var result *[]ApiInfo

	err := s.getForEntity(&result, "apis")
	if err != nil {
		return nil, err
	}

	return *result, nil
}

// GetAPIsByLabel retrieves a list of all APIs registered in Gravitee.
func (s *GraviteeSession) GetAPIsByLabel(label string) ([]ApiInfo, error) {
	result, err := s.GetAllAPIs()
	if err != nil {
		return nil, err
	}

	filtered := make([]ApiInfo, 0)

	for _, ai := range result {
		for _, lbl := range ai.Labels {
			if strings.EqualFold(lbl, label) {
				filtered = append(filtered, ai)
				break
			}
		}
	}

	return filtered, nil
}

// GetAPI retrieves details on an API registered in Gravitee.
func (s *GraviteeSession) GetAPI(id string) (*ApiDetails, error) {
	var result *ApiDetails

	err := s.getForEntity(&result, "apis", id)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// AddOrUpdateEndpoints adds or updates an endpoint to an API registered in Gravitee.
func (s *GraviteeSession) AddOrUpdateEndpoints(id string, endpoints []ApiDetailsEndpoint, replaceAll bool) error {
	res, err := s.getRaw("apis", id)
	if err != nil {
		return err
	}

	json, err := gabs.ParseJSON(res)
	if err != nil {
		return err
	}

	if replaceAll {
		json.Array("proxy", "endpoints")
	}

	for _, ep := range endpoints {
		err := s.addOrUpdateEndpointSingle(json, ep)
		if err != nil {
			return err
		}
	}

	// clean-up (PUT request will fail if the following properties are set)
	json.Delete("context_path")
	json.Delete("created_at")
	json.Delete("deployed_at")
	json.Delete("id")
	json.Delete("owner")
	json.Delete("picture_url")
	json.Delete("state")
	json.Delete("updated_at")

	return s.putRaw(json.Bytes(), "apis", id)
}

// AddOrUpdateEndpoint adds or updates an endpoint to an API registered in Gravitee.
func (s *GraviteeSession) addOrUpdateEndpointSingle(json *gabs.Container, endpoint ApiDetailsEndpoint) error {
	endpointsArray := json.Search("proxy", "endpoints")

	// update existing endpoint
	epcount, err := endpointsArray.ArrayCount()
	if err != nil {
		return err
	}

	updated := false

	for i := 0; i < epcount; i++ {
		ep, err := endpointsArray.ArrayElement(i)
		if err != nil {
			return err
		}

		if endpoint.Name == ep.Search("name").Data() {
			_, err = endpointsArray.SetIndex(endpoint, i)

			updated = true
			break
		}
	}

	// add new endpoint
	if !updated {
		err = json.ArrayAppend(endpoint, "proxy", "endpoints") // does not work on endpointsArray
		if err != nil {
			return err
		}
	}

	return nil
}

// GetAPIMetadata retrieves the metadata on an API registered in Gravitee.
func (s *GraviteeSession) GetAPIMetadata(id string) ([]ApiMetadata, error) {
	var result *[]ApiMetadata

	err := s.getForEntity(&result, "apis", id, "metadata")
	if err != nil {
		return nil, err
	}

	return *result, nil
}

// GetLocalAPIMetadata retrieves a local metadata entry on an API registered in Gravitee.
func (s *GraviteeSession) GetLocalAPIMetadata(id string, metadataKey string) (*ApiMetadata, error) {
	var result *ApiMetadata

	err := s.getForEntity(&result, "apis", id, "metadata", metadataKey)
	if err != nil {
		switch v := err.(type) {
		case RequestError:
			if v.HttpStatus == 404 {
				return nil, nil
			}
		}

		return nil, err
	}

	return result, nil
}

// UnsetLocalAPIMetadata removes a local metadata entry on an API registered in Gravitee.
func (s *GraviteeSession) UnsetLocalAPIMetadata(id string, metadataKey string) error {
	err := s.delete("apis", id, "metadata", metadataKey)
	if err != nil {
		switch v := err.(type) {
		case RequestError:
			if v.HttpStatus == 404 {
				return nil
			}
		}

		return err
	}

	return nil
}

// SetLocalAPIMetadata updates or creates a local metadata entry on an API registered in Gravitee.
func (s *GraviteeSession) SetLocalAPIMetadata(id string, metadataKey string, value string, format ApiMetadataFormat) error {
	req := ApiMetadata{
		Key:        metadataKey,
		Name:       metadataKey,
		LocalValue: value,
		Format:     format,
	}

	err := s.put(req, "apis", id, "metadata", metadataKey)
	if err != nil {
		switch v := err.(type) {
		case RequestError:
			if v.HttpStatus == 404 {
				return nil
			}
		}

		return err
	}

	return nil
}

// DeployAPI deploys the current configuration of the API to the gateway instances.
func (s *GraviteeSession) DeployAPI(id string) error {
	err := s.post("", "apis", id, "deploy")
	if err != nil {
		return err
	}

	return nil
}
