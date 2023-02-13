package ovirt

// OVirt engine API address/credentials
type Site struct {
	Url      string `json:"url" validate:"required,startswith=https://"` // "engine API address"
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Ca       string `json:"ca,omitempty"`
	Insecure bool   `json:"insecure,omitempty"`
}
