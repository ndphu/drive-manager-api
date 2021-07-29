package main

import (
	"drive-manager-api/controller"
	"drive-manager-api/dao"
	"drive-manager-api/entity"
	"drive-manager-api/middleware"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
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
	controller.ConfigController(api.Group("/config"))
	controller.SearchController(api.Group("/search"))
	controller.UserController(api.Group("/user"))

	controller.AdminController(api.Group("/admin"))
	controller.StreamController(api.Group("/stream"))

	manage := api.Group("/manage")
	manage.Use(middleware.FirebaseAuthMiddleware())
	controller.ProjectController(manage.Group("/"))
	controller.AccountController(manage.Group("/"))
	controller.ViewController(manage.Group("/view"))
	controller.SyncController(manage.Group("/sync"))

	//updateProjects()

	fmt.Println("Starting server")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "OK"})
	})
	r.Run()
}

func updateProjects() {
	accounts := make([]entity.DriveAccount, 0)
	dao.Collection("drive_account").Find(bson.M{
		"projectId": bson.ObjectIdHex("5c709c76a88fb50ed0843d4b"),
	}).All(&accounts)

	for _, account := range accounts {
		log.Println(account.Name)
		account.ProjectId = bson.ObjectIdHex("5c70a0eca88fb51da4b59611")
		if err := dao.Collection("drive_account").UpdateId(account.Id, &account); err != nil {
			log.Println("fail to update", account.Name)
		} else {
			log.Println("updated", account.Name)
		}
	}
}
