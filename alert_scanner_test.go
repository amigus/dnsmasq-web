package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newScannerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	db.Exec(testDatabaseSQL)
	assert.NoError(t, db.AutoMigrate(&Request{}, &Lease{}, &Client{}, &DeviceAnnotation{}, &Alert{}, &DeviceAlertState{}))
	return db
}

func TestScanAlertsBaselineCreatesNoAlerts(t *testing.T) {
	db := newScannerDB(t)

	assert.NoError(t, scanAlertsOnce(db))
	var alerts []Alert
	assert.NoError(t, db.Find(&alerts).Error)
	assert.Empty(t, alerts)

	var state DeviceAlertState
	assert.NoError(t, db.First(&state).Error)
	assert.True(t, state.Initialized)
	assert.Contains(t, state.BaselineMACs, "bc:32:b2:3b:13:d4")

	// Repeated scans over unchanged clients stay quiet.
	assert.NoError(t, scanAlertsOnce(db))
	assert.NoError(t, db.Find(&alerts).Error)
	assert.Empty(t, alerts)
}

func TestScanAlertsCreatesOneAlertPerNewMAC(t *testing.T) {
	db := newScannerDB(t)
	assert.NoError(t, scanAlertsOnce(db))

	assert.NoError(t, db.Exec("INSERT INTO clients (mac, hostname, client_id, vendor_class, updated) VALUES ('aa:bb:cc:dd:ee:01', 'new-device', '', '', '2026-10-12 12:00:00')").Error)

	assert.NoError(t, scanAlertsOnce(db))
	var alerts []Alert
	assert.NoError(t, db.Find(&alerts).Error)
	if assert.Len(t, alerts, 1) {
		assert.Equal(t, "aa:bb:cc:dd:ee:01", alerts[0].MAC)
		assert.Equal(t, "new-device", alerts[0].Hostname)
		assert.Equal(t, "", alerts[0].IPv4)
		assert.Nil(t, alerts[0].Acknowledged)
	}

	// Repeated scans do not duplicate the alert.
	assert.NoError(t, scanAlertsOnce(db))
	assert.NoError(t, db.Find(&alerts).Error)
	assert.Len(t, alerts, 1)
}

func TestScanAlertsSnapshotsActiveLeaseIPv4(t *testing.T) {
	db := newScannerDB(t)
	assert.NoError(t, scanAlertsOnce(db))

	assert.NoError(t, db.Exec("INSERT INTO clients (mac, hostname, client_id, vendor_class, updated) VALUES ('aa:bb:cc:dd:ee:02', 'leased-device', '', '', '2026-10-12 12:00:00')").Error)
	assert.NoError(t, db.Exec("INSERT INTO leases (mac, ipv4, added, renewed) VALUES ('aa:bb:cc:dd:ee:02', '192.168.1.200', '2026-10-12 12:00:00', '2026-10-12 12:30:00')").Error)

	assert.NoError(t, scanAlertsOnce(db))
	var alerts []Alert
	assert.NoError(t, db.Where("mac = ?", "aa:bb:cc:dd:ee:02").Find(&alerts).Error)
	if assert.Len(t, alerts, 1) {
		assert.Equal(t, "192.168.1.200", alerts[0].IPv4)
	}
}

func TestScanAlertsNormalizesMAC(t *testing.T) {
	db := newScannerDB(t)
	assert.NoError(t, scanAlertsOnce(db))

	assert.NoError(t, db.Exec("INSERT INTO clients (mac, hostname, client_id, vendor_class, updated) VALUES ('AA:BB:CC:DD:EE:03', 'uppercase-device', '', '', '2026-10-12 12:00:00')").Error)

	assert.NoError(t, scanAlertsOnce(db))
	var alerts []Alert
	assert.NoError(t, db.Find(&alerts).Error)
	if assert.Len(t, alerts, 1) {
		assert.Equal(t, "aa:bb:cc:dd:ee:03", alerts[0].MAC)
	}
}

func TestScanAlertsFailureDoesNotInitializeBaseline(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	// Migrate only the alert tables so the clients query fails.
	assert.NoError(t, db.AutoMigrate(&Alert{}, &DeviceAlertState{}))

	assert.Error(t, scanAlertsOnce(db))
	var state DeviceAlertState
	assert.Error(t, db.First(&state).Error)
}
