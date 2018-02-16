package zevenetlb

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"testing"
)

const (
	unitTestFarmName  = "UNITTESTGO"
	unitTestVirtualIP = "10.209.0.31"
)

func webGet(url string) (string, int, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	res, err := client.Get(url)

	if err != nil {
		return "", 0, err
	}

	defer res.Body.Close()

	// read response body
	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", 0, err
	}

	return string(data), res.StatusCode, nil
}

func TestGetAllFarms(t *testing.T) {
	session := createTestSession(t)

	res, err := session.GetAllFarms()

	if err != nil {
		t.Fatal(err)
	}

	if len(res) <= 0 {
		t.Fatal("No farms returned")
	}

	//t.Logf("Farms: %v", res)

	// get the farm details
	for _, f := range res {
		farm, err := session.GetFarm(f.FarmName)

		if err != nil {
			t.Fatal(err)
		}

		if farm == nil {
			t.Fatalf("Farm not found: %v", f.FarmName)
		}

		t.Logf("Farm: %v", farm)

		for _, c := range farm.Certificates {
			t.Logf("  Certificate: %v", c)
		}

		for _, s := range farm.Services {
			t.Logf("  Service: %v", s)

			for _, b := range s.Backends {
				t.Logf("    Backend: %v", b)
			}
		}
	}
}

func TestRoundtripHTTPFarm(t *testing.T) {
	session := createTestSession(t)

	// ensure the farm does not exist
	_, err := session.DeleteFarm(unitTestFarmName)

	if err != nil {
		t.Fatal(err)
	}

	// create the new farm
	farm, err := session.CreateFarmAsHTTP(unitTestFarmName, unitTestVirtualIP, 0)

	if err != nil {
		t.Fatal(err)
	}

	defer session.DeleteFarm(farm.FarmName)

	t.Logf("New farm: %v, Status: %v", farm, farm.Status)

	// update 503 message (for testing)
	farm.ErrorString503 = fmt.Sprintf("Service unavailable ## %v ##", rand.Int63())

	err = session.UpdateFarm(farm)

	if err != nil {
		t.Fatal(err)
	}

	// restart the farm
	err = session.RestartFarm(farm.FarmName)

	if err != nil {
		t.Fatal(err)
	}

	// try to connect
	resBody, resCode, err := webGet(fmt.Sprintf("http://%v:%v", farm.VirtualIP, farm.VirtualPort))

	if err != nil {
		t.Fatal(err)
	}

	// check if the return code matches
	if resCode != 503 {
		t.Fatalf("Expected HTTP status code 503, but got %v", resCode)
	}

	// check if the message matches
	if !strings.Contains(resBody, farm.ErrorString503) {
		t.Fatalf("Expected the status message to contain '%v', but got '%v'", farm.ErrorString503, resBody)
	}

	// add a service
	service, err := session.CreateService(farm.FarmName, "service1")

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Service: %v", service)

	// add a backend
	backend, err := session.CreateBackend(farm.FarmName, service.ServiceName, "176.58.123.25", 80)

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Backend: %v", backend)

	// restart the farm
	err = session.RestartFarm(farm.FarmName)

	if err != nil {
		t.Fatal(err)
	}

	// try to connect
	resBodyExpect, _, _ := webGet("http://176.58.123.25")
	resBody, resCode, err = webGet(fmt.Sprintf("http://%v:%v", farm.VirtualIP, farm.VirtualPort))

	if err != nil {
		t.Fatal(err)
	}

	// check if the return code matches
	if resCode != 200 {
		t.Fatalf("Expected HTTP status code 200, but got %v", resCode)
	}

	// check if the message matches
	if !strings.Contains(resBody, resBodyExpect) {
		t.Fatalf("Expected the status message to contain '%v', but got '%v'", resBodyExpect, resBody)
	}

	// enable maintenance
	err = session.SetBackendMaintenance(backend, true, true)

	if err != nil {
		t.Fatal(err)
	}

	// disable maintenance
	err = session.SetBackendMaintenance(backend, false, false)

	if err != nil {
		t.Fatal(err)
	}

	// cleaning up, delete the backend
	deleted, err := session.DeleteBackend(farm.FarmName, service.ServiceName, backend.ID)

	if err != nil {
		t.Fatal(err)
	}

	if !deleted {
		t.Fatal("Expected deleting the backend to succeed, but failed")
	}

	// delete the service
	deleted, err = session.DeleteService(farm.FarmName, service.ServiceName)

	if err != nil {
		t.Fatal(err)
	}

	if !deleted {
		t.Fatal("Expected deleting the service to succeed, but failed")
	}

	// done, delete the farm
	deleted, err = session.DeleteFarm(farm.FarmName)

	if err != nil {
		t.Fatal(err)
	}

	if !deleted {
		t.Fatal("Expected deleting the farm to succeed, but failed")
	}
}

func TestRoundtripHTTPSFarm(t *testing.T) {
	session := createTestSession(t)

	// ensure the farm does not exist
	_, err := session.DeleteFarm(unitTestFarmName)

	if err != nil {
		t.Fatal(err)
	}

	// retrieve certificate list
	certs, err := session.GetAllCertificates()

	if err != nil {
		t.Fatal(err)
	}

	if len(certs) <= 0 {
		t.Fatal("No certificates found on Zevenet loadbalancer")
	}

	certName := certs[0].Filename

	t.Logf("Using certificate: %v", certName)

	// create the new farm
	farm, err := session.CreateFarmAsHTTPS(unitTestFarmName, unitTestVirtualIP, 0, certName)

	if err != nil {
		t.Fatal(err)
	}

	defer session.DeleteFarm(unitTestFarmName)

	t.Logf("New farm: %v, Status: %v", farm, farm.Status)

	// update 503 message (for testing)
	farm.ErrorString503 = fmt.Sprintf("Service unavailable ## %v ##", rand.Int63())

	err = session.UpdateFarm(farm)

	if err != nil {
		t.Fatal(err)
	}

	// restart the farm
	err = session.RestartFarm(farm.FarmName)

	if err != nil {
		t.Fatal(err)
	}

	// try to connect
	resBody, resCode, err := webGet(fmt.Sprintf("https://%v:%v", farm.VirtualIP, farm.VirtualPort))

	if err != nil {
		t.Fatal(err)
	}

	// check if the return code matches
	if resCode != 503 {
		t.Fatalf("Expected HTTP status code 503, but got %v", resCode)
	}

	// check if the message matches
	if !strings.Contains(resBody, farm.ErrorString503) {
		t.Fatalf("Expected the status message to contain '%v', but got '%v'", farm.ErrorString503, resBody)
	}

	// add a service
	service, err := session.CreateService(farm.FarmName, "service1")

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Service: %v", service)

	// enable backend re-encryption
	service.EncryptedBackends = true

	err = session.UpdateService(service)

	if err != nil {
		t.Fatal(err)
	}

	// add a backend
	backend, err := session.CreateBackend(farm.FarmName, service.ServiceName, "176.58.123.25", 443)

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Backend: %v", backend)

	// restart the farm
	err = session.RestartFarm(farm.FarmName)

	if err != nil {
		t.Fatal(err)
	}

	// try to connect
	resBodyExpect, _, _ := webGet("http://176.58.123.25")
	resBody, resCode, err = webGet(fmt.Sprintf("https://%v:%v", farm.VirtualIP, farm.VirtualPort))

	if err != nil {
		t.Fatal(err)
	}

	// check if the return code matches
	if resCode != 200 {
		t.Fatalf("Expected HTTP status code 200, but got %v", resCode)
	}

	// check if the message matches
	if !strings.Contains(resBody, resBodyExpect) {
		t.Fatalf("Expected the status message to contain '%v', but got '%v'", resBodyExpect, resBody)
	}

	// cleaning up, delete the backend
	deleted, err := session.DeleteBackend(farm.FarmName, service.ServiceName, backend.ID)

	if err != nil {
		t.Fatal(err)
	}

	if !deleted {
		t.Fatal("Expected deleting the backend to succeed, but failed")
	}

	// delete the service
	deleted, err = session.DeleteService(farm.FarmName, service.ServiceName)

	if err != nil {
		t.Fatal(err)
	}

	if !deleted {
		t.Fatal("Expected deleting the service to succeed, but failed")
	}

	// done, delete the farm
	deleted, err = session.DeleteFarm(farm.FarmName)

	if err != nil {
		t.Fatal(err)
	}

	if !deleted {
		t.Fatal("Expected deleting to succeed, but failed")
	}
}
