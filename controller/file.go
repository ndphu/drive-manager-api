package controller

import (
	"github.com/gin-gonic/gin"
)

func FileController(r *gin.RouterGroup) {
	r.GET("/countByName/:name", func(c *gin.Context) {
		//user := CurrentUser(c)
		//if count, err := dao.FileIndex().Count(bson.M{
		//	"name":     c.Param("name"),
		//	"owner":    user.Id,
		//	"disabled": bson.M{"$ne": true},
		//}); err != nil {
		//	c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		//	return
		//} else {
		//	c.JSON(200, gin.H{"count": count})
		//}
		c.JSON(200, gin.H{"count": 0})
	})
}
