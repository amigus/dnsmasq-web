package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenChecker_Get(t *testing.T) {
	ttc := NewTokenChecker(1, 1, time.Hour)

	token1 := ttc.Get()
	assert.NotEmpty(t, token1, "Expected a non-empty token")
	token2 := ttc.Get()
	assert.Equal(t, token1, token2, "Expected the same token until it is used or expires")
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
	assert.Equal(t, token1, token4, "Expected the ring buffer to wrap to the first token")
}

func TestTokenChecker_GetTheSameTokenAgain(t *testing.T) {
	ttc := NewTokenChecker(1, 1, time.Second)

	token1 := ttc.Get()
	assert.NotEmpty(t, token1, "Expected a non-empty token")
	assert.True(t, ttc.Check(token1), "Expected the token to be accepted once")
	token2 := ttc.Get()
	assert.NotEqual(t, token1, token2, "Expected an exhausted token slot to be replaced")
}

func TestTokenChecker_Check(t *testing.T) {
	ttc := NewTokenChecker(1, 1, time.Hour)

	token1 := ttc.Get()
	assert.True(t, ttc.Check(token1), "Expected token to be valid")
}

func TestTokenChecker_CheckInvalid(t *testing.T) {
	ttc := NewTokenChecker(1, 1, time.Hour)
	assert.False(t, ttc.Check("not-a-real-token"), "Expected an unknown token to be invalid")
}

func TestTokenChecker_CheckReuse(t *testing.T) {
	ttc := NewTokenChecker(1, 2, time.Hour)
	token1 := ttc.Get()
	assert.True(t, ttc.Check(token1), "Expected token to be valid")
	assert.True(t, ttc.Check(token1), "Expected token to be valid for the configured number of uses")
	assert.False(t, ttc.Check(token1), "Expected token to be invalid after the reuse limit is reached")
}

func TestTokenChecker_CheckTooManyReuses(t *testing.T) {
	ttc := NewTokenChecker(1, 1, time.Hour)
	token1 := ttc.Get()
	assert.True(t, ttc.Check(token1), "Expected token to be valid")
	assert.False(t, ttc.Check(token1), "Expected token to be invalid")
	assert.NotEqual(t, token1, ttc.Get(), "Expected a fresh token after the first one is exhausted")
}

func TestTokenChecker_CheckAfterTimeout(t *testing.T) {
	ttc := NewTokenChecker(1, 2, time.Millisecond*2)
	token1 := ttc.Get()
	time.Sleep(time.Millisecond * 3)
	assert.False(t, ttc.Check(token1), "Expected the token to expire after the timeout")
}
