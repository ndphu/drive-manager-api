package controller

import (
	"github.com/gin-gonic/gin"
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
	//accountService := service.GetAccountService()

	r.POST("/fileUpload", func(c *gin.Context) {
		//user := CurrentUser(c)
		//var ur FileUploadRequest
		//if err := c.ShouldBindJSON(&ur); err != nil {
		//	c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
		//	return
		//}
		//var accounts []entity.DriveAccount
		//uploadBuffer := int64(3221223823) // 3GB
		//if err := dao.DriveAccount().Template(func(col *mongo.Collection) error {
		//	return col.Pipe([]bson.M{
		//		{
		//			"$match": bson.M{
		//
		//				"owner":     user.Id,
		//				"type":      "service_account",
		//				"disabled":  bson.M{"$ne": true},
		//				"available": bson.M{"$gt": ur.Size + uploadBuffer},
		//			},
		//		},
		//		{"$sample": bson.M{"size": 1}},
		//		{
		//			"$project": bson.M{
		//				"_id":       1,
		//				"key":       1,
		//				"projectId": 1,
		//				"usage":     1,
		//				"available": 1,
		//				"limit":     1,
		//			},
		//		}}).All(&accounts)
		//}); err != nil {
		//	c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		//	return
		//}
		//
		//for _, account := range accounts {
		//	if account.Limit-account.Usage > ur.Size {
		//		// pickup account
		//		token, err := accountService.GetAccessToken(&account)
		//		if err != nil {
		//			// going to next account
		//			continue
		//		}
		//		c.JSON(200, gin.H{
		//			"uploadInfo": UploadResponse{
		//				AccessToken: token,
		//				AccountId:   account.Id.Hex(),
		//			},
		//		})
		//		return
		//	}
		//}
		// TODO
		c.AbortWithStatusJSON(500, gin.H{"error": "cannot find suitable account for upload request"})
	})
}
