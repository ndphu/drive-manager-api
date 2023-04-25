package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Item struct {
	Id      primitive.ObjectID      `json:"id" bson:"_id"`
	Name    string             `json:"name"`
	Type    string             `json:"type"`
	Owner   primitive.ObjectID      `json:"owner"`
	File    *service.FileIndex `json:"file,omitempty" bson:"file,omitempty"`
	Parent  primitive.ObjectID      `json:"parent,omitempty" bson:"parent,omitempty"`
	Deleted bool               `json:"deleted" bson:"deleted"`
}

type FileInfo struct {
	FileId    string `json:"fileId"`
	AccountId string `json:"accountId"`
	ParentId  string `json:"parentId"`
}

func BrowseController(r *gin.RouterGroup) {
	//as := service.GetAccountService()
	r.POST("/item/:itemId/files", func(c *gin.Context) {
		parentId := c.Param("itemId")
		var file service.FileIndex
		if err := c.ShouldBindJSON(&file); err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
			return
		}

		item := Item{
			Id:    bson.NewObjectId(),
			Type:  "file",
			Name:  file.Name,
			Owner: file.Owner,
			File:  &file,
		}
		if !(parentId == "root" || parentId == "") {
			item.Parent = primitive.ObjectIDFromHex(parentId)
		}

		if err := dao.Item().Insert(item); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"item": item, "success": true})
	})

	r.POST("/item/:itemId/folders", func(c *gin.Context) {
		var item Item
		if err := c.ShouldBindJSON(&item); err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
			return
		}

		parentId := c.Param("itemId")
		if bson.IsObjectIDFromHex(parentId) {
			item.Parent = primitive.ObjectIDFromHex(parentId)
		}

		item.Owner = CurrentUser(c).Id
		item.Type = "folder"
		item.Id = bson.NewObjectId()

		if err := dao.Item().Insert(item); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"item": item, "success": true})
	})

	r.GET("/item/:itemId", func(c *gin.Context) {
		u := CurrentUser(c)
		parentId := c.Param("itemId")

		var condition = bson.M{
			"owner":   u.Id,
			"deleted": bson.M{"$ne": true},
		}
		if bson.IsObjectIDFromHex(parentId) {
			condition["parent"] = primitive.ObjectIDFromHex(parentId)
		} else {
			condition["parent"] = nil
		}
		var items []Item
		//items := make([]Item, 0)
		if err := dao.Item().Find(condition, &items); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		} else {
			c.JSON(200, gin.H{"success": true, "items": items})
		}
	})

	r.DELETE("/item/:itemId", func(c *gin.Context) {
		u := CurrentUser(c)
		itemId := c.Param("itemId")
		if !bson.IsObjectIDFromHex(itemId) {
			c.AbortWithStatusJSON(400, gin.H{"success": false})
			return
		}

		var i Item
		if err := dao.Item().Find(bson.M{
			"_id":   primitive.ObjectIDFromHex(itemId),
			"owner": u.Id,
		}, &i); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
			return
		}
		i.Deleted = true
		if err := dao.Item().UpdateId(primitive.ObjectIDFromHex(itemId), i); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"success": true})
	})
}
