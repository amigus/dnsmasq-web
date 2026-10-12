package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTokenCheckerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ttc := NewTokenChecker(1, 2, time.Hour)

	validToken := ttc.Get()
	r = TokenCheckerHeader(r, ttc, "X-Token")

	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Token", validToken)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected status OK for valid token")
}

func TestTokenCheckerHeaderRejectsExhaustedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ttc := NewTokenChecker(1, 1, time.Hour)
	validToken := ttc.Get()
	r = TokenCheckerHeader(r, ttc, "X-Token")

	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Token", validToken)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected the first use to succeed")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Token", validToken)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Expected an exhausted token to be rejected")
}

func TestTokenCheckerHeaderWithInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ttc := NewTokenChecker(1, 1, time.Hour)

	invalidToken := uuid.NewString()
	r = TokenCheckerHeader(r, ttc, "X-Token")

	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Token", invalidToken)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Expected status OK for valid token")
}

func TestTokenCheckerPublisher(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ttc := NewTokenChecker(1, 5, time.Minute)

	r = TokenCheckerPublisher(r, ttc, "/")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected status OK")
	assert.NotEmpty(t, w.Body.String(), "Expected a non-empty token in response")
	if err := uuid.Validate(w.Body.String()); err != nil {
		t.Error("Expected a valid UUID in response", err)
	}
}

func TestTokenCheckerHeaderCookieFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ttc := NewTokenChecker(1, 2, time.Hour)
	validToken := ttc.Get()
	r = TokenCheckerHeader(r, ttc, "X-Token")

	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: dashboardCookieName, Value: validToken})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected the dashboard cookie to authenticate")
}

func TestTokenCheckerHeaderLoginExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ttc := NewTokenChecker(1, 1, time.Hour)
	r = TokenCheckerHeader(r, ttc, "X-Token")

	r.GET(dashboardLoginPath, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, dashboardLoginPath, nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected the login route to be exempt from token checking")
}
