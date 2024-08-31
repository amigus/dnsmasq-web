package main

import (
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TokenChecker interface {
	Get() string
	Check(token string) bool
}

// token struct to hold UUID, count, and time of expiration
type token struct {
	uuid       string
	count      int
	expiration time.Time
}

type tokenChecker struct {
	tokens   []token
	maxCount int
	maxUses  int
	maxTime  time.Duration
	index    int
	mu       sync.Mutex
}

func NewTokenChecker(maxCount, maxUses int, maxTime time.Duration) TokenChecker {
	tokens := make([]token, maxCount)
	for i := range tokens {
		tokens[i] = token{
			uuid:       uuid.New().String(),
			count:      0,
			expiration: time.Now().Add(maxTime),
		}
	}
	if maxUses <= 0 {
		maxUses = math.MaxInt
	}
	return &tokenChecker{
		tokens:   tokens,
		maxCount: maxCount,
		maxUses:  maxUses,
		maxTime:  maxTime,
		index:    0,
	}
}

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

func TokenCheckerHeader(r *gin.Engine, ttc TokenChecker, headerName string) *gin.Engine {
	r.Use(func(c *gin.Context) {
		if ttc.Check(c.GetHeader(headerName)) {
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized token"})
		}
	})

	return r
}

func TokenCheckerPublisher(r *gin.Engine, ttc TokenChecker) *gin.Engine {
	r.GET("/", func(c *gin.Context) {c.String(http.StatusOK, ttc.Get())})

	return r
}
