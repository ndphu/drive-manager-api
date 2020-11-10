package controller

import (
	"github.com/gin-gonic/gin"
	"drive-manager-api/dao"
	"drive-manager-api/entity"
	"drive-manager-api/middleware"
)

func AdminController(r *gin.RouterGroup) error {
	r.Use(middleware.FirebaseAuthMiddleware(), middleware.AdminFilter())

	r.GET("/users", func(c *gin.Context) {
		users := make([]entity.User, 0)
		err := dao.Collection("user").Find(nil).All(&users)
		if err != nil {
			ServerError("Fail to query users", err, c)
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
