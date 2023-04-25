package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/service"
	"go.mongodb.org/mongo-driver/bson"
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
			Id:    primitive.NewObjectID(),
			Type:  "file",
			Name:  file.Name,
			Owner: file.Owner,
			File:  &file,
		}
		if !(parentId == "root" || parentId == "") {
			hex, _ := primitive.ObjectIDFromHex(parentId)
			item.Parent = hex
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
		if primitive.IsValidObjectID(parentId) {
			hex, _ := primitive.ObjectIDFromHex(parentId)
			item.Parent = hex
		}

		item.Owner = CurrentUser(c).Id
		item.Type = "folder"
		item.Id = primitive.NewObjectID()

		if err := dao.Item().Insert(item); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"item": item, "success": true})
	})

	r.GET("/item/:itemId", func(c *gin.Context) {
		u := CurrentUser(c)
		parentId := c.Param("itemId")

		var condition = bson.D{
			{"owner", u.Id},
			{"deleted", bson.D{{"$ne", true}}},
		}
		if primitive.IsValidObjectID(parentId) {
			hex, _ := primitive.ObjectIDFromHex(parentId)
			condition.Map()["parent"] = hex
		} else {
			condition.Map()["parent"] = nil
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
		if !primitive.IsValidObjectID(itemId) {
			c.AbortWithStatusJSON(400, gin.H{"success": false})
			return
		}

		var i Item
		itemIdHex, _ := primitive.ObjectIDFromHex(itemId)
		if err := dao.Item().Find(bson.D{
			{"_id", itemIdHex},
			{"owner", u.Id},
		}, &i); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
			return
		}
		i.Deleted = true
		// TODO
		//hexId, _ := primitive.ObjectIDFromHex(itemId)
		//if err := dao.Item().Update(hexId, i); err != nil {
		//	c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
		//	return
		//}
		c.JSON(200, gin.H{"success": true})
	})
}
