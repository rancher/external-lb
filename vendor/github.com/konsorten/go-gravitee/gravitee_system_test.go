package gravitee

import (
	"os"
	"strings"
	"testing"
)

func createTestSession(t *testing.T) *GraviteeSession {
	return createTestSessionEx(t, "", "")
}

func createTestSessionEx(t *testing.T, username, password string) *GraviteeSession {
	// retrieve api auth if undefined
	if username == "" {
		username = os.Getenv("GRAVITEE_USER")

		if username == "" {
			username = "admin"
		}
	}

	if password == "" {
		password = os.Getenv("GRAVITEE_PWD")

		if password == "" {
			password = "admin"
		}
	}

	// retrieve hostname
	host := os.Getenv("GRAVITEE_HOSTNAME")

	if host == "" {
		host = "api.konsorten-api.de"
	}

	// create the session
	session, err := Connect(host, username, password, nil)

	if err != nil {
		t.Fatalf("Failed to connect to Gravitee Management API: %v", err)
	}

	return session
}

func TestInvalidHost(t *testing.T) {
	_, err := Connect("d0esn0tex1st", "inva1dus3r", "inval1dAp1K3y", nil)

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
