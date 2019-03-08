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
		err := dao.Collection("user").Find(nil).All(&users)
		if err != nil {
			ServerError("Fail to query users", err, c)
			return
		}
		c.JSON(200, users)
	})

	return nil
}
