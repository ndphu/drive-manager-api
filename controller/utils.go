package controller

import (
	"github.com/gin-gonic/gin"
	"drive-manager-api/entity"
)

func CurrentUser(c*gin.Context) *entity.User {
	val, _ := c.Get("user")
	user := val.(*entity.User)
	return user
}
