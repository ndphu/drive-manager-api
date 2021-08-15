package controller

import (
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
)

type ProjectTreeView struct {
	Id          bson.ObjectId     `json:"id" bson:"_id"`
	DisplayName string            `json:"displayName" bson:"displayName"`
	ProjectId   string            `json:"projectId" bson:"projectId"`
	Owner       bson.ObjectId     `json:"owner" bson:"owner"`
	Accounts    []AccountTreeView `json:"accounts" bson:"accounts"`
}

type AccountTreeView struct {
	Id        bson.ObjectId `json:"id,omitempty" bson:"_id"`
	AccountId int64         `json:"accountId,omitempty" bson:"accountId"`
	Name      string        `json:"name,omitempty" bson:"name"`
}

func ViewController(r *gin.RouterGroup) {
	r.Use(middleware.FirebaseAuthMiddleware())
	r.GET("/tree/projects", func(c *gin.Context) {
		user := CurrentUser(c)
		projects := make([]ProjectTreeView, 0)
		if err := dao.Project().Pipe([]bson.M{
			{
				"$match": bson.M{"owner": user.Id},
			},
			{
				"$lookup": bson.M{
					"from": "drive_account",
					"let":  bson.M{"projectId": "$_id"},
					"pipeline": []bson.M{
						{"$match": bson.M{
							"$expr": bson.M{
								"$eq": []string{"$projectId", "$$projectId"},
							},
						}},
						{"$project": bson.M{
							"_id":  1,
							"name": 1,
						}},
					},
					"as": "accounts",
				},
			},
			{
				"$project": bson.M{
					"_id":         1,
					"displayName": 1,
					"accounts":    1,
				},
			},
		}, &projects); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"projects": projects})
		}
	})
}
