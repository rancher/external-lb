package zevenetlb

import (
	"testing"
)

func TestGetAllCertificates(t *testing.T) {
	session := createTestSession(t)

	res, err := session.GetAllCertificates()

	if err != nil {
		t.Fatal(err)
	}

	if len(res) <= 0 {
		t.Fatal("No certificates returned")
	}

	//t.Logf("Certificates: %v", res)

	for _, c := range res {
		t.Logf("Certificate: %v", c)
	}
}
