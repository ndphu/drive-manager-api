package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/middleware"
)

func AdminController(r *gin.RouterGroup) error {
	r.Use(middleware.FirebaseAuthMiddleware(), middleware.AdminFilter())

	r.GET("/users", func(c *gin.Context) {
		users := make([]entity.User, 0)
		err := dao.User().FindAll(&users)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(200, users)
	})

	r.POST("/users/sync", func(c *gin.Context) {

	})

	r.POST("/user/:id", func(c *gin.Context) {

	})

	r.PUT("/user/:id", func(c *gin.Context) {

	})

	return nil
}
