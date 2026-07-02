package extension

// Info describes extension identity shown in /meta and admin defaults.
type Info struct {
	Name          string
	Version       string
	UpdateURLBase string
	AdminMenuName string
}

type metaResponse struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	UpdateURLBase string `json:"update_url_base"`
	RouteHash     string `json:"route_hash,omitempty"`
}

type capabilitiesResponse struct {
	Capabilities map[string]string `json:"capabilities"`
	Defaults     capabilityDefaults  `json:"defaults"`
}

type capabilityDefaults struct {
	Admin adminDefaults `json:"admin"`
}

type adminDefaults struct {
	MenuName string `json:"menu_name"`
}
