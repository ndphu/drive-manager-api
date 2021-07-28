package middleware

import (
	"github.com/gin-gonic/gin"
	"drive-manager-api/entity"
	"log"
)

func AdminFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(401, gin.H{"err": "No Login User Found"})
		} else {
			if IsAdmin(user.(*entity.User)) {
				c.Next()
			} else {
				c.AbortWithStatusJSON(403, gin.H{"err": "You are not admin"})
			}
		}
	}
}

func IsAdmin(user *entity.User) bool {
	log.Println("checking user roles", user.Roles)
	for _, role := range user.Roles {
		if role == "admin" {
			return true
		}
	}
	return false
}
