package zevenetlb

import (
	"fmt"
	"strings"
)

type systemVersionResponse struct {
	Description string        `json:"description"`
	Params      SystemVersion `json:"params"`
}

// SystemVersion contains information about the system version.
// See https://www.zevenet.com/zapidoc_ce_v3.1/#show-version
type SystemVersion struct {
	ApplianceVersion string `json:"appliance_version"`
	Hostname         string `json:"hostname"`
	KernelVersion    string `json:"kernel_version"`
	SystemDate       string `json:"system_date"`
	ZevenetVersion   string `json:"zevenet_version"`
}

// String returns the version number of the system, e.g. "ZCE 5 (v5.0)"
func (sv *SystemVersion) String() string {
	return fmt.Sprintf("%v (v%v)", sv.ApplianceVersion, sv.ZevenetVersion)
}

// IsCommunityEdition checks if the Zevenet loadbalancer is the Community Edition (vs Enterprise Edition)
func (sv *SystemVersion) IsCommunityEdition() bool {
	return strings.HasPrefix(sv.ApplianceVersion, "ZCE")
}

// GetSystemVersion returns system version information.
func (s *ZapiSession) GetSystemVersion() (*SystemVersion, error) {
	var result *systemVersionResponse

	err := s.getForEntity(&result, "system", "version")

	if err != nil {
		return nil, err
	}

	return &result.Params, nil
}
