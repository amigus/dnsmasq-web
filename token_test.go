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

func TestTokenChecker_Get(t *testing.T) {
	ttc := &tokenChecker{
		tokens:   make([]token, 1),
		maxUses:  1,
		maxTime:  time.Hour,
		maxCount: 1,
	}

	token1 := ttc.Get()
	assert.NotEmpty(t, token1, "Expected a non-empty token")
	assert.Equal(t, 0, ttc.index, "Expected index to be zero after token1")
	assert.Equal(t, 1, ttc.tokens[0].count, "Expected token count to be zero")
	token2 := ttc.Get()
	assert.NotEqual(t, token1, token2, "Expected a different token")
	assert.Equal(t, 0, ttc.index, "Expected index to be zero after token2")
	assert.Equal(t, 1, ttc.tokens[0].count, "Expected token count to be reset")
}

func TestTokenChecker_GetAfterTimeout(t *testing.T) {
	ttc := NewTokenChecker(1, 2, time.Millisecond*2)

	token1 := ttc.Get()
	assert.NotEmpty(t, token1, "Expected a non-empty token")
	token2 := ttc.Get()
	assert.Equal(t, token1, token2, "Expected the same token")
	time.Sleep(time.Millisecond * 3)
	token3 := ttc.Get()
	assert.NotEqual(t, token1, token3, "Expected a different token")
}
func TestTokenChecker_GetTooManyTokens(t *testing.T) {
	ttc := NewTokenChecker(3, 1, time.Second)

	token1 := ttc.Get()
	assert.NotEmpty(t, token1, "Expected a non-empty token")
	token2 := ttc.Get()
	assert.NotEqual(t, token1, token2, "Expected a different token")
	token3 := ttc.Get()
	assert.NotEqual(t, token2, token3, "Expected a different token")
	token4 := ttc.Get()
	assert.NotEqual(t, token3, token4, "Expected a different token")
}

func TestTokenChecker_GetTheSameTokenAgain(t *testing.T) {
	ttc := NewTokenChecker(3, 2, time.Second)

	token1 := ttc.Get()
	assert.NotEmpty(t, token1, "Expected a non-empty token")
	token2 := ttc.Get()
	assert.NotEqual(t, token1, token2, "Expected a different token")
	token3 := ttc.Get()
	assert.NotEqual(t, token2, token3, "Expected a different token")
	token4 := ttc.Get()
	assert.Equal(t, token4, token1, "Expected token4 to match token1")
}

func TestTokenChecker_Check(t *testing.T) {
	ttc := NewTokenChecker(1, 1, time.Hour)

	token1 := ttc.Get()
	assert.True(t, ttc.Check(token1), "Expected token to be valid")
}

func TestTokenChecker_CheckInvalid(t *testing.T) {
	ttc := &tokenChecker{
		tokens:   make([]token, 1),
		maxUses:  1,
		maxTime:  time.Hour,
		maxCount: 1,
	}
	token1 := ttc.Get()
	assert.True(t, ttc.Check(token1), "Expected token to be valid")
	ttc.tokens[0].uuid = uuid.NewString()
	assert.False(t, ttc.Check(token1), "Expected token to be invalid")
}

func TestTokenChecker_CheckReuse(t *testing.T) {
	ttc := NewTokenChecker(1, 2, time.Hour)
	token1 := ttc.Get()
	assert.True(t, ttc.Check(token1), "Expected token to be valid")
	token2 := ttc.Get()
	assert.False(t, ttc.Check(token1), "Expected token to be invalid")
	assert.True(t, ttc.Check(token2), "Expected token to be valid")
}

func TestTokenChecker_CheckTooManyReuses(t *testing.T) {
	ttc := NewTokenChecker(1, 1, time.Hour)
	token1 := ttc.Get()
	assert.True(t, ttc.Check(token1), "Expected token to be valid")
	token2 := ttc.Get()
	assert.False(t, ttc.Check(token1), "Expected token to be invalid")
	assert.True(t, ttc.Check(token2), "Expected token to be valid")
}

func TestTokenCheckerHeader(t *testing.T) {
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
	assert.Equal(t, http.StatusOK, w.Code, "Expected status OK for valid token")
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
	ttc := &tokenChecker{
		tokens:   make([]token, 1),
		maxUses:  5,
		maxTime:  time.Minute,
		maxCount: 1,
	}

	r = TokenCheckerPublisher(r, ttc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected status OK")
	assert.NotEmpty(t, w.Body.String(), "Expected a non-empty token in response")
	if err := uuid.Validate(w.Body.String()); err != nil {
		t.Error("Expected a valid UUID in response", err)
	}
}
