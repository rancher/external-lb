package zevenetlb

import (
	"os"
	"strings"
	"testing"
)

func createTestSession(t *testing.T) *ZapiSession {
	return createTestSessionEx(t, "")
}

func createTestSessionEx(t *testing.T, apiKey string) *ZapiSession {
	// retrieve api key if undefined
	if apiKey == "" {
		apiKey = os.Getenv("ZAPI_KEY")

		if apiKey == "" {
			t.Fatal("Failed to retrieve ZAPI key from environment variable ZAPI_KEY")
		}
	}

	// retrieve host name
	host := os.Getenv("ZAPI_HOSTNAME")

	if host == "" {
		host = "lb002.konsorten.net:444"
	}

	// create the session
	session, err := Connect(host, apiKey, nil)

	if err != nil {
		t.Fatalf("Failed to connect to Zevenet API: %v", err)
	}

	return session
}

func TestInvalidHost(t *testing.T) {
	_, err := Connect("d0esn0tex1st", "inval1dAp1K3y", nil)

	if err == nil {
		t.Fatal("Error expected")
	}

	if !strings.Contains(err.Error(), "no such host") {
		t.Fatalf("Wrong error message returned: %v", err)
	}
}

func TestPing(t *testing.T) {
	session := createTestSession(t)

	success, msg := session.Ping()

	if !success {
		t.Fatalf("Ping failed: %v", msg)
	}
}

func TestGetSystemVersion(t *testing.T) {
	session := createTestSession(t)

	res, err := session.GetSystemVersion()

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Version: %v", res)
}

func TestIsCommunityEdition(t *testing.T) {
	session := createTestSession(t)

	res, err := session.GetSystemVersion()

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Is Community Edition: %v", res.IsCommunityEdition())
}
