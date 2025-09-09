package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TokenCheckerHeader adds a middleware to the gin engine that requires a valid token in the given header.
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

// TokenCheckerPublisher adds a route to the gin engine for GET on the given path that returns a valid token.
func TokenCheckerPublisher(r *gin.Engine, ttc TokenChecker, path string) *gin.Engine {
	r.GET(path, func(c *gin.Context) { c.String(http.StatusOK, ttc.Get()) })

	return r
}
