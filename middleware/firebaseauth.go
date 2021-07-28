package middleware

import (
	"drive-manager-api/service"
	"github.com/gin-gonic/gin"
	"strings"
)

func FirebaseAuthMiddleware() gin.HandlerFunc {
	authService, _ := service.GetAuthService()
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader,"Bearer ")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			c.AbortWithStatusJSON(401, gin.H{"errors": "Missing JWT Token"})
		} else {
			user, err := authService.GetUserFromToken(token)
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{"err": err})
			} else {
				c.Set("user", user)
				c.Set("jwtToken", token)
				c.Next()
			}
		}
	}

}
