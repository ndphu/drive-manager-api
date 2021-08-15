package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/middleware"
	"github.com/ndphu/drive-manager-api/service"
	"sync"
)

func SearchController(r *gin.RouterGroup) error {
	r.Use(middleware.FirebaseAuthMiddleware())
	r.GET("quickSearch", func(c *gin.Context) {
		user := CurrentUser(c)
		query := c.Query("query")
		files := make([]service.FileIndex, 0)
		accounts := make([]entity.DriveAccount, 0)
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			dao.FileIndex().Template(func(col *mgo.Collection) error {
				return col.Find(bson.M{
					"owner": user.Id,
					"name":  bson.RegEx{Pattern: query, Options: "i"},
				}).Limit(20).All(&files)
			})
		}()

		go func() {
			defer wg.Done()
			dao.DriveAccount().Template(func(col *mgo.Collection) error {
				return col.Find(bson.M{
					"name":  bson.RegEx{Pattern: query, Options: "i"},
					"owner": user.Id,
				}).
					Select(bson.M{
						"_id":  1,
						"name": 1,
					}).
					Limit(20).
					All(&accounts)
			})

		}()

		wg.Wait()

		c.JSON(200, gin.H{"files": files, "accounts": accounts})
	})
	return nil
}
