package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"sync"
)

func SearchController(r *gin.RouterGroup) error {
	r.GET("quickSearch", func(c *gin.Context) {
		query := c.Query("query")
		files := make([]entity.DriveFile, 0)
		accounts := make([]entity.DriveAccount, 0)
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			dao.Collection("file_entry").Find(bson.M{
				"name": bson.RegEx{Pattern: query, Options: "i"},
			}).Limit(20).All(&files)
		}()

		go func() {
			defer wg.Done()
			dao.Collection("drive_account").
				Find(bson.M{
					"name": bson.RegEx{Pattern: query, Options: "i"},
				}).
				Select(bson.M{
					"_id":  1,
					"name": 1,
				}).
				Limit(20).
				All(&accounts)
		}()

		wg.Wait()

		c.JSON(200, gin.H{"files": files, "accounts": accounts})
	})
	return nil
}
