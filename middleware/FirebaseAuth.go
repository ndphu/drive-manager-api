package middleware

import (
	"github.com/gin-gonic/gin"
	"drive-manager-api/service"
	"log"
	"strings"
)

func FirebaseAuthMiddleware() gin.HandlerFunc {
	authService, _ := service.GetAuthService()
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader,"Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(401, gin.H{"errors": "Missing JWT Token"})
		} else {
			log.Println("JWT Token", token)
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
