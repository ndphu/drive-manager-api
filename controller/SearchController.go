package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
)

func SearchController(r *gin.RouterGroup) error {
	r.GET("", func(c *gin.Context) {
		query := c.Query("query")
		files := make([]entity.DriveFile, 0)
		dao.Collection("file").Find(bson.M{
			"name": bson.RegEx{Pattern: query, Options: "i"},
		}).Limit(20).All(&files)
		c.JSON(200, files)
	})
	return nil
}
