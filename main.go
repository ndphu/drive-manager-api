package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/controller"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"log"
)

func main() {
	r := gin.Default()

	c := cors.DefaultConfig()
	c.AllowAllOrigins = true
	c.AllowCredentials = true
	c.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	c.AllowHeaders = []string{"Origin", "Authorization", "Content-Type", "Content-Length", "X-Requested-With", "Authorization"}

	r.Use(cors.New(c))

	api := r.Group("/api")
	controller.AccountController(api.Group("/manage/driveAccount"))
	controller.SearchController(api.Group("/search"))
	controller.UserController(api.Group("/user"))
	controller.ProjectController(api.Group("/project"))
	controller.AdminController(api.Group("/admin"))

	//updateProjects()

	fmt.Println("Starting server")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "OK"})
	})
	r.Run()
}

func updateProjects()  {
	accounts := make([]entity.DriveAccount, 0)
	dao.Collection("drive_account").Find(bson.M{
		"projectId": bson.ObjectIdHex("5c709c76a88fb50ed0843d4b"),
	}).All(&accounts)

	for _, account := range accounts {
		log.Println(account.Name)
		account.ProjectId = bson.ObjectIdHex("5c70a0eca88fb51da4b59611")
		if err:=dao.Collection("drive_account").UpdateId(account.Id, &account); err != nil {
			log.Println("fail to update", account.Name)
		} else {
			log.Println("updated", account.Name)
		}
	}
}
