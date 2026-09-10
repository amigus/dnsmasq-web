package client

// Lease represents an active DHCP lease.
type Lease struct {
	Mac         string `json:"mac"`
	Hostname    string `json:"hostname"`
	ClientID    string `json:"client_id"`
	VendorClass string `json:"vendor_class"`
	IPv4        string `json:"ipv4"`
	Added       string `json:"added"`
	Renewed     string `json:"renewed"`
}

// ClientInfo represents a DHCP client with request history.
type ClientInfo struct {
	Mac         string   `json:"mac"`
	Hostname    string   `json:"hostname"`
	ClientID    string   `json:"client_id"`
	VendorClass string   `json:"vendor_class"`
	Updated     string   `json:"updated"`
	Requests    int      `json:"requests"`
	IPv4s       []string `json:"ipv4s"`
}

// IPHistory represents the history of an IP address for a given MAC.
type IPHistory struct {
	IPv4             string `json:"ipv4"`
	FirstSeen        string `json:"first_seen"`
	LastSeen         string `json:"last_seen"`
	RequestedOptions string `json:"requested_options"`
	Hostname         string `json:"hostname"`
	VendorClass      string `json:"vendor_class"`
}

// MacHistory represents the history of a MAC address for a given IPv4.
type MacHistory struct {
	Mac              string `json:"mac"`
	FirstSeen        string `json:"first_seen"`
	LastSeen         string `json:"last_seen"`
	RequestedOptions string `json:"requested_options"`
	Hostname         string `json:"hostname"`
	VendorClass      string `json:"vendor_class"`
}

// Request represents a DHCP request event.
type Request struct {
	Received         string `json:"received"`
	Argument         string `json:"argument"`
	Mac              string `json:"mac"`
	IPv4             string `json:"ipv4"`
	ClientID         string `json:"client_id"`
	RequestedOptions string `json:"requested_options"`
}

// Reservation represents a DHCP reservation in dnsmasq.
type Reservation struct {
	MAC       string   `json:"mac"`
	IPv4      string   `json:"ipv4"`
	Hostname  string   `json:"hostname"`
	Tags      []string `json:"tags"`
	LeaseTime string   `json:"lease_time,omitempty"`
}

// RequestCount represents a request count breakdown by IP.
type RequestCount struct {
	IPv4        string `json:"ipv4"`
	Mac         string `json:"mac"`
	Hostname    string `json:"hostname"`
	VendorClass string `json:"vendor_class"`
	Count       int    `json:"count"`
}
