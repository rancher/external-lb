package zevenetlb

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"
)

var defaultConfigOptions = &ConfigOptions{
	APICallTimeout: 60 * time.Second,
	ZapiVersion:    "3.1",
}

// ConfigOptions contains some advanced settings on server communication.
type ConfigOptions struct {
	APICallTimeout time.Duration
	ZapiVersion    string
}

// ZapiSession is a container for our session state.
type ZapiSession struct {
	Host          string
	ZapiKey       string
	Transport     *http.Transport
	ConfigOptions *ConfigOptions
}

// String returns the session's hostname.
func (s *ZapiSession) String() string {
	return s.Host
}

// APIRequest builds our request before sending it to the server.
type APIRequest struct {
	Method      string
	URL         string
	Body        string
	ContentType string
}

// RequestError contains information about any error we get from a request.
type RequestError struct {
	Message     string `json:"message,omitempty"`
	Description string `json:"description,omitempty"`
}

// Error returns the error message.
func (r *RequestError) Error() error {
	if r.Description != "" {
		return fmt.Errorf("%v failed: %v", r.Description, r.Message)
	}

	return fmt.Errorf("%v", r.Message)
}

// Connect sets up our connection to the Zevenet system.
func Connect(host, zapiKey string, configOptions *ConfigOptions) (*ZapiSession, error) {
	var url string
	if !strings.HasPrefix(host, "http") {
		url = fmt.Sprintf("https://%s", host)
	} else {
		url = host
	}
	if configOptions == nil {
		configOptions = defaultConfigOptions
	}

	// create the session
	session := &ZapiSession{
		Host:    url,
		ZapiKey: zapiKey,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		ConfigOptions: configOptions,
	}

	// initialize the session
	err := session.initialize()

	if err != nil {
		return nil, err
	}

	// done
	return session, nil
}

func (s *ZapiSession) initialize() (err error) {
	// test connection
	_, err = s.GetSystemVersion()
	return
}

// Ping checks if the loadbalancer is available.
func (s *ZapiSession) Ping() (bool, string) {
	// test connection
	_, err := s.GetSystemVersion()

	if err != nil {
		return false, err.Error()
	}

	return true, ""
}

// apiCall is used to query the ZAPI.
func (s *ZapiSession) apiCall(options *APIRequest) ([]byte, error) {
	var req *http.Request
	client := &http.Client{
		Transport: s.Transport,
		Timeout:   s.ConfigOptions.APICallTimeout,
	}
	url := fmt.Sprintf("%v/zapi/v%v/zapi.cgi/%v", s.Host, s.ConfigOptions.ZapiVersion, options.URL)
	body := bytes.NewReader([]byte(options.Body))
	req, _ = http.NewRequest(strings.ToUpper(options.Method), url, body)

	req.Header.Set("ZAPI_KEY", s.ZapiKey)

	// fmt.Println("REQ -- ", options.Method, " ", url, " -- ", options.Body)

	if len(options.ContentType) > 0 {
		req.Header.Set("Content-Type", options.ContentType)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode >= 400 {
		if res.Header["Content-Type"][0] == "application/json" {
			return data, s.checkError(data)
		}

		return data, fmt.Errorf("HTTP %d :: %s", res.StatusCode, string(data[:]))
	}

	// fmt.Println("Resp --", res.StatusCode, " -- ", string(data))
	return data, nil
}

func (s *ZapiSession) iControlPath(parts []string) string {
	var buffer bytes.Buffer
	for i, p := range parts {
		buffer.WriteString(strings.Replace(p, "/", "~", -1))
		if i < len(parts)-1 {
			buffer.WriteString("/")
		}
	}
	return buffer.String()
}

//Generic delete
func (s *ZapiSession) delete(path ...string) error {
	req := &APIRequest{
		Method: "delete",
		URL:    s.iControlPath(path),
	}

	_, callErr := s.apiCall(req)
	return callErr
}

func (s *ZapiSession) post(body interface{}, path ...string) error {
	marshalJSON, err := jsonMarshal(body)
	if err != nil {
		return err
	}

	req := &APIRequest{
		Method:      "post",
		URL:         s.iControlPath(path),
		Body:        strings.TrimRight(string(marshalJSON), "\n"),
		ContentType: "application/json",
	}

	_, callErr := s.apiCall(req)
	return callErr
}

func (s *ZapiSession) put(body interface{}, path ...string) error {
	marshalJSON, err := jsonMarshal(body)
	if err != nil {
		return err
	}

	req := &APIRequest{
		Method:      "put",
		URL:         s.iControlPath(path),
		Body:        strings.TrimRight(string(marshalJSON), "\n"),
		ContentType: "application/json",
	}

	_, callErr := s.apiCall(req)
	return callErr
}

//Get a url and populate an entity. If the entity does not exist (404) then the
//passed entity will be untouched and false will be returned as the second parameter.
//You can use this to distinguish between a missing entity or an actual error.
func (s *ZapiSession) getForEntity(e interface{}, path ...string) error {
	req := &APIRequest{
		Method:      "get",
		URL:         s.iControlPath(path),
		ContentType: "application/json",
	}

	resp, err := s.apiCall(req)
	if err != nil {
		var reqError RequestError
		json.Unmarshal(resp, &reqError)
		return err
	}

	err = json.Unmarshal(resp, e)
	if err != nil {
		return err
	}

	return nil
}

// checkError handles any errors we get from our API requests. It returns either the
// message of the error, if any, or nil.
func (s *ZapiSession) checkError(resp []byte) error {
	if len(resp) == 0 {
		return nil
	}

	var reqError RequestError

	err := json.Unmarshal(resp, &reqError)
	if err != nil {
		return fmt.Errorf("%s\n%s", err.Error(), string(resp[:]))
	}

	err = reqError.Error()
	if err != nil {
		return err
	}

	return nil
}

// jsonMarshal specifies an encoder with 'SetEscapeHTML' set to 'false' so that <, >, and & are not escaped. https://golang.org/pkg/encoding/json/#Marshal
// https://stackoverflow.com/questions/28595664/how-to-stop-json-marshal-from-escaping-and
func jsonMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

// Helper to copy between transfer objects and model objects to hide the myriad of boolean representations
// in the iControlREST api. DTO fields can be tagged with bool:"yes|enabled|true" to set what true and false
// marshal to.
func marshal(to, from interface{}) error {
	toVal := reflect.ValueOf(to).Elem()
	fromVal := reflect.ValueOf(from).Elem()
	toType := toVal.Type()
	for i := 0; i < toVal.NumField(); i++ {
		toField := toVal.Field(i)
		toFieldType := toType.Field(i)
		fromField := fromVal.FieldByName(toFieldType.Name)
		if fromField.Interface() != nil && fromField.Kind() == toField.Kind() {
			toField.Set(fromField)
		} else if toField.Kind() == reflect.Bool && fromField.Kind() == reflect.String {
			switch fromField.Interface() {
			case "yes", "enabled", "true":
				toField.SetBool(true)
				break
			case "no", "disabled", "false", "":
				toField.SetBool(false)
				break
			default:
				return fmt.Errorf("Unknown boolean conversion for %s: %s", toFieldType.Name, fromField.Interface())
			}
		} else if fromField.Kind() == reflect.Bool && toField.Kind() == reflect.String {
			tag := toFieldType.Tag.Get("bool")
			switch tag {
			case "yes":
				toField.SetString(toBoolString(fromField.Interface().(bool), "yes", "no"))
				break
			case "enabled":
				toField.SetString(toBoolString(fromField.Interface().(bool), "enabled", "disabled"))
				break
			case "true":
				toField.SetString(toBoolString(fromField.Interface().(bool), "true", "false"))
				break
			}
		} else {
			return fmt.Errorf("Unknown type conversion %s -> %s", fromField.Kind(), toField.Kind())
		}
	}
	return nil
}

func toBoolString(b bool, trueStr, falseStr string) string {
	if b {
		return trueStr
	}
	return falseStr
}
