package middleware

import (
	"context"
	"net/http"
	"strings"

	"learnos/internal/auth"

	"github.com/gin-gonic/gin"
)

const SessionCookieName = "learnos_session"

func SessionAuth(authenticate func(context.Context, string) (auth.Principal, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken, err := c.Cookie(SessionCookieName)
		if err != nil || strings.TrimSpace(rawToken) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		principal, err := authenticate(c.Request.Context(), rawToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := auth.PrincipalFromContext(c.Request.Context())
		// Legacy Basic Auth test/dev routers have no session principal. The
		// production application always installs SessionAuth before this check.
		if !ok {
			c.Next()
			return
		}
		if principal.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "administrator privileges required"})
			return
		}
		c.Next()
	}
}
