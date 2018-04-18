package gravitee

type UserReference struct {
	Id          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type configTenantsResponse struct {
}

// Ping checks if the API is available.
func (s *GraviteeSession) Ping() (bool, error) {
	var result *[]configTenantsResponse

	err := s.getForEntity(&result, "configuration", "tenants")

	if err != nil {
		return false, err
	}

	return true, nil
}
