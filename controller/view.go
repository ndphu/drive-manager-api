package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProjectTreeView struct {
	Id          primitive.ObjectID `json:"id" bson:"_id"`
	DisplayName string             `json:"displayName" bson:"displayName"`
	ProjectId   string             `json:"projectId" bson:"projectId"`
	Owner       primitive.ObjectID `json:"owner" bson:"owner"`
	Accounts    []AccountTreeView  `json:"accounts" bson:"accounts"`
}

type AccountTreeView struct {
	Id        primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	AccountId int64              `json:"accountId,omitempty" bson:"accountId"`
	Name      string             `json:"name,omitempty" bson:"name"`
}

func ViewController(r *gin.RouterGroup) {
	r.Use(middleware.FirebaseAuthMiddleware())
	r.GET("/tree/projects", func(c *gin.Context) {
		user := CurrentUser(c)
		projects := make([]ProjectTreeView, 0)
		if cursor, err := dao.Project().Aggregate(context.Background(), mongo.Pipeline{
			{
				{"$match", bson.D{{"owner", user.Id}}},
			},
			{
				{"$lookup", bson.D{
					{"from", "drive_account"},
					{"let", bson.D{{"projectId", "$_id"}}},
					{"pipeline", mongo.Pipeline{
						{{"$match", bson.D{
							{"$expr", bson.D{
								{"$eq", []string{"$projectId", "$$projectId"}},
							}},
						}}},
						{
							{"$project", bson.D{
								{"_id", 1},
								{"name", 1},
							}},
						},
						{
							{"$sort", bson.D{
								{"_id", 1},
								{"disabled", -1},
							}},
						},
					}},
					{"as", "accounts"},
				}},
			},
			{
				{"$sort", bson.D{
					{"disabled", 1},
					{"_id", 1},
				}},
			},
			{
				{"$project", bson.D{
					{"_id", 1},
					{"displayName", 1},
					{"accounts", 1},
				}},
			},
		}); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else if err := cursor.All(context.Background(), &projects); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"projects": projects})
		}
	})
}
