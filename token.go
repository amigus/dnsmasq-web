package main

import (
	"time"

	"github.com/amigus/dnsmasq-web/internal/auth"
)

type TokenChecker = auth.TokenChecker

// NewTokenChecker creates a new TokenChecker with the given parameters.
// The maxCount is the number of tokens to allocate at once.
// The maxUses is the maximum number of times each token can be used before it expires.
// The timeout is the aount of time before each token expires.
// Examples:
// Issue 10 tokens that can be used fifteen times each for up to a minute.
// NewTokenChecker(10, 15, time.Minute)
// Issue 5 tokens that can be used 100 times each for up to 3 hours.
// NewTokenChecker(5, 100, 3*time.Hour)
// Issue 3 tokens that can be used an unlimited number of times for up to 8 hours.
// NewTokenChecker(3, 0, 8*time.Hour)
// Issue 1 token that can be used an unlimited number of times forever.
// NewTokenChecker(1, 0, 0)
func NewTokenChecker(maxCount, maxUses int, timeout time.Duration) TokenChecker {
	return auth.NewTokenChecker(maxCount, maxUses, timeout)
}
