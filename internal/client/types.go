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

// AddressHistory represents the assignment history for an IPv4 address.
type AddressHistory struct {
	IPv4        string       `json:"ipv4"`
	Assignments []MacHistory `json:"assignments"`
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

// DeviceAnnotation represents application-owned device metadata.
type DeviceAnnotation struct {
	Name  string   `json:"name"`
	Owner string   `json:"owner"`
	Notes string   `json:"notes"`
	Tags  []string `json:"tags"`
}

// AnnotationPatch carries optional annotation fields for a PATCH request.
// Nil fields are omitted; empty strings and slices clear stored values.
type AnnotationPatch struct {
	Name  *string   `json:"name,omitempty"`
	Owner *string   `json:"owner,omitempty"`
	Notes *string   `json:"notes,omitempty"`
	Tags  *[]string `json:"tags,omitempty"`
}

// DeviceObservation represents DHCP-observed facts for a device.
type DeviceObservation struct {
	Hostname     string `json:"hostname"`
	ClientID     string `json:"client_id"`
	VendorClass  string `json:"vendor_class"`
	FirstSeen    string `json:"first_seen"`
	LastSeen     string `json:"last_seen"`
	RequestCount int    `json:"request_count"`
}

// DeviceLease represents an active lease held by a device.
type DeviceLease struct {
	IPv4    string `json:"ipv4"`
	Added   string `json:"added"`
	Renewed string `json:"renewed"`
}

// Device represents a canonical device keyed by MAC address.
type Device struct {
	MAC          string            `json:"mac"`
	Observed     DeviceObservation `json:"observed"`
	ActiveLeases []DeviceLease     `json:"active_leases"`
	Annotation   *DeviceAnnotation `json:"annotation,omitempty"`
}

// DeviceAddress represents a historical IPv4 assignment for a device.
type DeviceAddress struct {
	IPv4      string `json:"ipv4"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

// Alert represents a persistent new-device alert.
type Alert struct {
	ID           uint    `json:"id"`
	MAC          string  `json:"mac"`
	Hostname     string  `json:"hostname"`
	IPv4         string  `json:"ipv4"`
	DetectedAt   string  `json:"detected_at"`
	Acknowledged *string `json:"acknowledged_at"`
}
