package auth

import (
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TokenChecker interface {
	Get() string
	Check(token string) bool
}

type token struct {
	uuid       string
	count      int
	expiration time.Time
}

type Checker struct {
	tokens   []token
	maxCount int
	maxUses  int
	maxTime  time.Duration
	index    int
	mu       sync.Mutex
}

func newToken(timeout time.Duration) token {
	t := token{uuid: uuid.New().String()}
	if timeout > 0 {
		t.expiration = time.Now().Add(timeout)
	}
	return t
}

func (ttc *Checker) tokenExpired(token token) bool {
	return !token.expiration.IsZero() && time.Now().After(token.expiration)
}

func NewTokenChecker(maxCount, maxUses int, timeout time.Duration) TokenChecker {
	if maxCount < 1 {
		maxCount = 1
	}
	tokens := make([]token, maxCount)
	for i := range tokens {
		tokens[i] = newToken(timeout)
	}
	if maxUses <= 0 {
		maxUses = math.MaxInt
	}
	return &Checker{
		tokens:   tokens,
		maxCount: maxCount,
		maxUses:  maxUses,
		maxTime:  timeout,
	}
}

func (ttc *Checker) Get() string {
	ttc.mu.Lock()
	defer ttc.mu.Unlock()

	token := &ttc.tokens[ttc.index]
	if token.uuid == "" || ttc.tokenExpired(*token) || token.count >= ttc.maxUses {
		*token = newToken(ttc.maxTime)
	}
	ttc.index = (ttc.index + 1) % ttc.maxCount
	return token.uuid
}

func (ttc *Checker) Check(token string) bool {
	ttc.mu.Lock()
	defer ttc.mu.Unlock()

	for i, t := range ttc.tokens {
		if t.uuid == token {
			if ttc.tokenExpired(t) || t.count >= ttc.maxUses {
				return false
			}
			ttc.tokens[i].count++
			return true
		}
	}
	return false
}