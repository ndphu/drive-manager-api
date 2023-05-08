package controller

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/middleware"
	"github.com/ndphu/drive-manager-api/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"sync"
)

func SearchController(r *gin.RouterGroup) error {
	r.Use(middleware.FirebaseAuthMiddleware())
	r.GET("quickSearch", func(c *gin.Context) {
		user := CurrentUser(c)
		query := c.Query("query")
		fmt.Println(user, query)
		files := make([]service.FileIndex, 0)
		accounts := make([]entity.DriveAccount, 0)
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			if cursor, err := dao.FileIndex().Find(context.Background(), bson.D{
				{"owner", user.Id},
				{"name", bson.D{
					{"$regex", primitive.Regex{Pattern: query, Options: "i"}},
				}},
			}, options.Find().SetLimit(20)); err != nil {
				log.Println("Fail to search file with pattern:", query, "by error", err.Error())
			} else {
				if err := cursor.All(context.Background(), &files); err != nil {
					log.Println("Fail to parse file_index result by error", err.Error())
				}
			}
		}()

		go func() {
			defer wg.Done()
			if cursor, err := dao.DriveAccount().Find(context.Background(), bson.D{
				{"owner", user.Id},
				{"name", bson.D{
					{"$regex", primitive.Regex{Pattern: query, Options: "i"}},
				}},
			}, options.Find().SetLimit(20)); err != nil {
				log.Println("Fail to search drive accounts with pattern:", query, "by error", err.Error())
			} else {
				if err := cursor.All(context.Background(), &accounts); err != nil {
					log.Println("Fail to parse drive_account result by error", err.Error())
				}
			}
		}()

		wg.Wait()

		c.JSON(200, gin.H{"files": files, "accounts": accounts})
	})
	return nil
}
