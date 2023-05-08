package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type FileUploadRequest struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}

type UploadResponse struct {
	AccessToken string `json:"accessToken"`
	AccountId   string `json:"accountId"`
}

func UploadController(r *gin.RouterGroup) {
	accountService := service.GetAccountService()

	r.POST("/fileUpload", func(c *gin.Context) {
		user := CurrentUser(c)
		var ur FileUploadRequest
		if err := c.ShouldBindJSON(&ur); err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
			return
		}
		var accounts []entity.DriveAccount
		uploadBuffer := int64(3221223823) // 3GB
		if cursor, err := dao.DriveAccount().Aggregate(context.Background(), mongo.Pipeline{
			{
				{"$match", bson.D{
					{"owner", user.Id},
					{"type", "service_account"},
					{"disabled", bson.D{{"$ne", true}}},
					{"available", bson.D{{"$gt", ur.Size + uploadBuffer}}},
				},
				}},
			{{"$sample", bson.D{{"size", 1}}}},
			{{
				"$project", bson.D{
					{"_id", 1},
					{"key", 1},
					{"projectId", 1},
					{"usage", 1},
					{"available", 1},
					{"limit", 1},
				},
			}}}); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		} else if err := cursor.All(context.Background(), &accounts); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}

		for _, account := range accounts {
			if account.Limit-account.Usage > ur.Size {
				// pickup account
				token, err := accountService.GetAccessToken(&account)
				if err != nil {
					// going to next account
					continue
				}
				c.JSON(200, gin.H{
					"uploadInfo": UploadResponse{
						AccessToken: token,
						AccountId:   account.Id.Hex(),
					},
				})
				return
			}
		}
		c.AbortWithStatusJSON(500, gin.H{"error": "cannot find suitable account for upload request"})
	})
}
