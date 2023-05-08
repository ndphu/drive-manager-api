package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Favorite struct {
	FileId    string `json:"fileId"`
	AccountId string `json:"accountId"`
}

type FavoriteResult struct {
	Id   primitive.ObjectID `json:"id" bson:"_id"`
	File *service.FileIndex `json:"file,omitempty" bson:"file,omitempty"`
}

func HomeController(r *gin.RouterGroup) {
	r.GET("/favorite", func(c *gin.Context) {
		u := CurrentUser(c)
		var fr []FavoriteResult
		if cursor, err := dao.FileFavorite().Aggregate(context.Background(), mongo.Pipeline{
			{
				{"$match", bson.D{{"userId", u.Id}}},
			},
			{
				{"$lookup", bson.D{
					{"from", "file_index"},
					{"localField", "fileId"},
					{"foreignField", "fileId"},
					{"as", "files"},
				}},
			},
			{
				{"$project", bson.D{
					{"file", bson.D{{"$arrayElemAt", []interface{}{"$files", 0}}},
					},
				}},
			},
			{
				{"$match", bson.D{
					{"file", bson.D{
						{"$exists", true},
						{"$ne", nil},
					}},
				}},
			},
		}); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else if err := cursor.All(context.Background(), &fr); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"success": true, "favorites": fr})
		}
	})
}
