package main

import (
	"math"
	"net/netip"

	"gorm.io/gorm"
)

// poolResponse is the subnet-wide utilization report.
type poolResponse struct {
	Subnet      string  `json:"subnet"`
	Total       int     `json:"total"`
	Used        int     `json:"used"`
	Free        int     `json:"free"`
	Utilization float64 `json:"utilization"`
}

// poolReport derives subnet-wide usable-host totals from current leases.
// Utilization is subnet-wide, not dnsmasq dhcp-range scoped.
func poolReport(db *gorm.DB, subnet string) (poolResponse, error) {
	prefix, err := netip.ParsePrefix(subnet)
	if err != nil {
		return poolResponse{}, err
	}
	prefix = prefix.Masked()

	hostBits := 32 - prefix.Bits()
	var total int
	if hostBits <= 1 {
		// RFC 3021 point-to-point: a /31 has two usable addresses, a /32 one.
		total = 1 << hostBits
	} else {
		total = (1 << hostBits) - 2
	}

	var leases []Lease
	if err := db.Find(&leases).Error; err != nil {
		return poolResponse{}, err
	}
	used := make(map[string]bool, len(leases))
	for _, lease := range leases {
		addr, err := netip.ParseAddr(lease.IPv4)
		if err != nil || !addr.Is4() || !prefix.Contains(addr) {
			continue
		}
		used[addr.String()] = true
	}

	utilization := 0.0
	if total > 0 {
		utilization = math.Round(float64(len(used))*1000/float64(total)) / 10
	}
	return poolResponse{
		Subnet:      prefix.String(),
		Total:       total,
		Used:        len(used),
		Free:        total - len(used),
		Utilization: utilization,
	}, nil
}
