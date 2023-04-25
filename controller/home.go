package controller

import (
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/service"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
)

type Favorite struct {
	FileId    string `json:"fileId"`
	AccountId string `json:"accountId"`
}

type FavoriteResult struct {
	Id   primitive.ObjectID      `json:"id" bson:"_id"`
	File *service.FileIndex `json:"file,omitempty" bson:"file,omitempty"`
}

func HomeController(r *gin.RouterGroup) {
	r.GET("/favorite", func(c *gin.Context) {
		u := CurrentUser(c)
		var fr []FavoriteResult
		if err := dao.FileFavorite().Pipe([]bson.M{
			{
				"$match": bson.M{"userId": u.Id},
			},
			{
				"$lookup": bson.M{
					"from":         "file_index",
					"localField":   "fileId",
					"foreignField": "fileId",
					"as":           "files",
				},
			},
			{
				"$project": bson.M{
					"file": bson.M{"$arrayElemAt": []interface{}{"$files", 0},
					},
				},
			},
			{
				"$match": bson.M{
					"file": bson.M{
						"$exists": true,
						"$ne":     nil,
					},
				},
			},
		}, &fr); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"success": true, "favorites": fr})
		}
	})
}
