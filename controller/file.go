package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
)

func FileController(r *gin.RouterGroup) {
	r.GET("/countByName/:name", func(c *gin.Context) {
		user := CurrentUser(c)
		session, collection := dao.CollectionWithSession("file_index")
		defer session.Close()
		if count, err := collection.Find(bson.M{
			"name":  c.Param("name"),
			"owner": user.Id,
		}).Count(); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		} else {
			c.JSON(200, gin.H{"count": count})
		}
	})
}
