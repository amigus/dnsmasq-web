package main

import (
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TokenChecker is a trivial token manager. It issues tokens and checks their validity.
type TokenChecker interface {
	// Get returns a valid token subject to the maxUses and maxTime constraints.
	Get() string
	Check(token string) bool
}

// token struct to holds the token, a UUID, the (use) count, and time of expiration.
type token struct {
	uuid       string
	count      int
	expiration time.Time
}

// tokenChecker struct represents a "ring buffer."
// It holds the list of tokens, parameters, an index and a mutux.
type tokenChecker struct {
	tokens   []token
	maxCount int
	maxUses  int
	maxTime  time.Duration
	index    int
	mu       sync.Mutex
}

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
	tokens := make([]token, maxCount)
	for i := range tokens {
		tokens[i] = token{
			uuid:       uuid.New().String(),
			count:      0,
			expiration: time.Now().Add(timeout),
		}
	}
	// "Unlimited" is really just the maximum integer value...
	if maxUses <= 0 {
		maxUses = math.MaxInt
	}
	return &tokenChecker{
		tokens:   tokens,
		maxCount: maxCount,
		maxUses:  maxUses,
		maxTime:  timeout,
		index:    0,
	}
}

// Get returns a valid token subject to the maxUses and maxTime constraints.
func (ttc *tokenChecker) Get() string {
	ttc.mu.Lock()
	defer ttc.mu.Unlock()

	token := &ttc.tokens[ttc.index]
	token.count++
	if ttc.maxTime > 0 && time.Now().After(token.expiration) || token.count > ttc.maxUses {
		token.count = 1
		token.expiration = time.Now().Add(ttc.maxTime)
		token.uuid = uuid.New().String()
	}
	ttc.index = (ttc.index + 1) % ttc.maxCount
	return token.uuid
}

// Check returns true if the token is valid after incrementing the counter on it.
func (ttc *tokenChecker) Check(token string) bool {
	ttc.mu.Lock()
	defer ttc.mu.Unlock()

	for i, t := range ttc.tokens {
		if t.uuid == token {
			ttc.tokens[i].count++
			return true
		}
	}
	return false
}
