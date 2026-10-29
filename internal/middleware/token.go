package middleware

import (
	"context"
	"net/http"
	"strings"

	"learnos/internal/auth"

	"github.com/gin-gonic/gin"
)

// TokenOrSession authenticates a capture request either with a bearer API token
// or with the ordinary session cookie, so the same endpoint serves the plugin and
// the web UI without weakening either path.
//
// A bearer credential is never combined with the cookie: when the header is
// present it is the only accepted proof.
func TokenOrSession(
	authenticateSession func(context.Context, string) (auth.Principal, error),
	authenticateToken func(context.Context, string) (auth.Principal, error),
) gin.HandlerFunc {
	session := SessionAuth(authenticateSession)
	return func(c *gin.Context) {
		if raw := bearerToken(c); raw != "" {
			principal, err := authenticateToken(c.Request.Context(), raw)
			if err != nil {
				unauthorized(c)
				return
			}
			c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), principal))
			c.Next()
			return
		}
		session(c)
	}
}

// RequireScope rejects principals whose credential does not carry the scope.
// Session principals (nil scopes) pass: their limits come from role checks.
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := auth.PrincipalFromContext(c.Request.Context())
		if !ok {
			unauthorized(c)
			return
		}
		if !principal.Allows(scope) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "SCOPE_REQUIRED", "message": "该凭证没有这个操作权限。", "retryable": false},
			})
			return
		}
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(header) < 7 || !strings.EqualFold(header[:7], "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{"code": "UNAUTHORIZED", "message": "认证失败，请检查凭证是否有效。", "retryable": false},
	})
}
