package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDashboardRouter(t *testing.T, maxUses int) (*gin.Engine, TokenChecker) {
	t.Helper()
	r := gin.Default()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.Exec(testDatabaseSQL)
	r = LeaseDatabase(r, db, "192.168.1.0/24", 0)
	ttc := NewTokenChecker(1, maxUses, time.Hour)
	r = TokenCheckerHeader(r, ttc, tokenHeader)
	r = Dashboard(r, db, "192.168.1.0/24", "", ttc)
	return r, ttc
}

func dashboardLogin(t *testing.T, r *gin.Engine, ttc TokenChecker) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	form := strings.NewReader("token=" + ttc.Get())
	req, _ := http.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	return w
}

func TestDashboardRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, _ := setupDashboardRouter(t, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDashboardLoginRenders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, _ := setupDashboardRouter(t, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/login", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "dnsmasq-web login")
}

func TestDashboardLoginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, ttc := setupDashboardRouter(t, 0)

	w := dashboardLogin(t, r, ttc)
	assert.Equal(t, http.StatusSeeOther, w.Code)
	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, dashboardCookieName, cookies[0].Name)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)

	// The cookie grants access to the dashboard.
	w = httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookies[0])
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "dnsmasq-web dashboard")
	assert.Contains(t, w.Body.String(), "44:4f:8e:ce:fa:64")
}

func TestDashboardLoginInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, _ := setupDashboardRouter(t, 0)

	w := httptest.NewRecorder()
	form := strings.NewReader("token=not-a-real-token")
	req, _ := http.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

func TestDashboardLoginDoesNotConsumeUse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, ttc := setupDashboardRouter(t, 1)

	w := dashboardLogin(t, r, ttc)
	assert.Equal(t, http.StatusSeeOther, w.Code)
	cookie := w.Result().Cookies()[0]

	// A single-use token still works for the dashboard after login.
	w = httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, ttc := setupDashboardRouter(t, 0)

	w := dashboardLogin(t, r, ttc)
	cookie := w.Result().Cookies()[0]

	w = httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(cookie)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, -1, w.Result().Cookies()[0].MaxAge)
}

func TestDashboardWithoutSubnetShowsNotice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.Exec(testDatabaseSQL)
	r = LeaseDatabase(r, db, "", 0)
	ttc := NewTokenChecker(1, 0, time.Hour)
	r = TokenCheckerHeader(r, ttc, tokenHeader)
	r = Dashboard(r, db, "", "", ttc)

	w := dashboardLogin(t, r, ttc)
	cookie := w.Result().Cookies()[0]

	w = httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "not configured; start the server with -s")
}
