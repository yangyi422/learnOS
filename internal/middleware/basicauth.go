package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func BasicAuth(username, passwordHash string) gin.HandlerFunc {
	if username == "" && passwordHash == "" {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		providedUsername, providedPassword, ok := c.Request.BasicAuth()
		usernameMatches := subtle.ConstantTimeCompare([]byte(providedUsername), []byte(username)) == 1
		passwordMatches := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(providedPassword)) == nil

		if !ok || !usernameMatches || !passwordMatches {
			c.Header("WWW-Authenticate", `Basic realm="LearnOS", charset="UTF-8"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			return
		}

		c.Next()
	}
}
