package main

import (
	"encoding/json"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeviceAlertState is the singleton row holding the alert baseline: the set
// of client MACs that existed when alerting was first enabled.
type DeviceAlertState struct {
	ID           uint `gorm:"primaryKey"`
	Initialized  bool
	BaselineMACs string
}

// TableName pins the singleton table name.
func (DeviceAlertState) TableName() string {
	return "device_alert_state"
}

// startAlertScanner scans immediately and then on every tick of interval
// until the process exits. Scan failures are logged and retried on the next
// tick without advancing the alert baseline.
func startAlertScanner(db *gorm.DB, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if err := scanAlertsOnce(db); err != nil {
				log.Printf("alert scan failed: %v", err)
			}
			<-ticker.C
		}
	}()
}

// scanAlertsOnce performs one new-device alert scan. The first successful
// scan records the client MACs as the baseline and creates no alerts; later
// scans create exactly one alert per MAC outside the baseline and the alert
// table, snapshotting the observed hostname and any active leased IPv4.
func scanAlertsOnce(db *gorm.DB) error {
	var state DeviceAlertState
	if err := db.First(&state).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	var clients []Client
	if err := db.Find(&clients).Error; err != nil {
		return err
	}

	if !state.Initialized {
		macs := make([]string, 0, len(clients))
		for _, client := range clients {
			if parsed, err := validateMAC(client.Mac); err == nil && parsed.ToAddressString().IsValid() {
				macs = append(macs, parsed.ToColonDelimitedString())
			}
		}
		encoded, err := json.Marshal(macs)
		if err != nil {
			return err
		}
		return db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&DeviceAlertState{ID: 1, Initialized: true, BaselineMACs: string(encoded)}).Error
	}

	var baseline []string
	if state.BaselineMACs != "" {
		if err := json.Unmarshal([]byte(state.BaselineMACs), &baseline); err != nil {
			return err
		}
	}
	known := make(map[string]bool, len(baseline))
	for _, mac := range baseline {
		known[mac] = true
	}

	var alertedMACs []string
	if err := db.Model(&Alert{}).Pluck("mac", &alertedMACs).Error; err != nil {
		return err
	}
	for _, mac := range alertedMACs {
		known[mac] = true
	}

	var leases []Lease
	if err := db.Find(&leases).Error; err != nil {
		return err
	}
	activeByMAC := make(map[string]string, len(leases))
	for _, lease := range leases {
		if _, ok := activeByMAC[lease.Mac]; !ok {
			activeByMAC[lease.Mac] = lease.IPv4
		}
	}

	detectedAt := time.Now().UTC()
	for _, client := range clients {
		mac := client.Mac
		if parsed, err := validateMAC(mac); err == nil && parsed.ToAddressString().IsValid() {
			mac = parsed.ToColonDelimitedString()
		} else {
			continue
		}
		if known[mac] {
			continue
		}
		alert := Alert{
			MAC:        mac,
			Hostname:   client.Hostname,
			IPv4:       activeByMAC[client.Mac],
			DetectedAt: detectedAt,
		}
		if err := db.Create(&alert).Error; err != nil {
			return err
		}
		known[mac] = true
	}
	return nil
}
