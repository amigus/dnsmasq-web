package main

import (
	"testing"
	"time"

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
