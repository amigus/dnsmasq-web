package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// dashboardCookieName is the cookie that carries the dashboard session token.
const dashboardCookieName = "dnsmasq_web_token"

// dashboardLoginPath is the dashboard login route, exempt from token checking.
const dashboardLoginPath = "/login"

// dashboardLogoutPath is the dashboard logout route, exempt from token checking.
const dashboardLogoutPath = "/logout"

// TokenCheckerHeader adds a middleware to the gin engine that requires a valid token in the given header.
// Browser sessions may present the dashboard cookie instead. The dashboard login and logout routes are exempt.
func TokenCheckerHeader(r *gin.Engine, ttc TokenChecker, headerName string) *gin.Engine {
	r.Use(func(c *gin.Context) {
		if c.FullPath() == dashboardLoginPath || c.FullPath() == dashboardLogoutPath {
			c.Next()
			return
		}
		if ttc.Check(c.GetHeader(headerName)) {
			c.Next()
			return
		}
		if cookie, err := c.Cookie(dashboardCookieName); err == nil && ttc.Check(cookie) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized token"})
	})

	return r
}

// TokenCheckerPublisher adds a route to the gin engine for GET on the given path that returns a valid token.
func TokenCheckerPublisher(r *gin.Engine, ttc TokenChecker, path string) *gin.Engine {
	r.GET(path, func(c *gin.Context) {
		c.String(http.StatusOK, ttc.Get())
	})

	return r
}
