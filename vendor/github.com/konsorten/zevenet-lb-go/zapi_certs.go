package zevenetlb

import "fmt"

type certListResponse struct {
	Description string               `json:"description"`
	Params      []CertificateDetails `json:"params"`
}

// CertificateDetails contains the details on a certificate.
// See https://www.zevenet.com/zapidoc_ce_v3.1/#list-all-certificates
type CertificateDetails struct {
	CommonName     string `json:"CN"`
	CreationDate   string `json:"creation"`
	ExpirationDate string `json:"expiration"`
	Filename       string `json:"file"`
	Issuer         string `json:"issuer"`
	Type           string `json:"type"`
}

// String returns the certificates common name and filename.
func (sv CertificateDetails) String() string {
	return fmt.Sprintf("%v (%v)", sv.CommonName, sv.Filename)
}

// GetAllCertificates returns list of all available certificates and CSRs.
func (s *ZapiSession) GetAllCertificates() ([]CertificateDetails, error) {
	var result *certListResponse

	err := s.getForEntity(&result, "certificates")

	if err != nil {
		return nil, err
	}

	return result.Params, nil
}
