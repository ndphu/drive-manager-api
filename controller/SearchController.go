package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"drive-manager-api/dao"
	"drive-manager-api/entity"
	"drive-manager-api/middleware"
	"sync"
)

func SearchController(r *gin.RouterGroup) error {
	r.Use(middleware.FirebaseAuthMiddleware())
	r.GET("quickSearch", func(c *gin.Context) {
		query := c.Query("query")
		files := make([]entity.DriveFile, 0)
		accounts := make([]entity.DriveAccount, 0)
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			dao.Collection("file").Find(bson.M{
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
