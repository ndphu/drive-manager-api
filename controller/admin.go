package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/middleware"
	"go.mongodb.org/mongo-driver/bson"
)

func AdminController(r *gin.RouterGroup) error {
	r.Use(middleware.FirebaseAuthMiddleware(), middleware.AdminFilter())

	r.GET("/users", func(c *gin.Context) {
		users := make([]entity.User, 0)
		if cursor, err := dao.User().Find(context.Background(), bson.D{}); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
			return
		} else {
			if err := cursor.All(context.Background(), &users); err != nil {
				c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
			}
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
