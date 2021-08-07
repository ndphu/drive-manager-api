package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/controller"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/middleware"
	"github.com/ndphu/drive-manager-api/service"
	"log"
)

func main() {

	r := gin.Default()

	c := cors.DefaultConfig()
	c.AllowAllOrigins = true
	c.AllowCredentials = true
	c.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	c.AllowHeaders = []string{"Origin", "Authorization", "Content-Type", "Content-Length", "X-Requested-With", "Authorization", "X-Config-Api-Key"}

	//doSync()

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
	controller.HomeController(manage.Group("/home"))
	controller.UploadController(manage.Group("/upload"))
	controller.BrowseController(manage.Group("/browse"))
	controller.FileController(manage.Group("/file"))

	//updateProjects()

	fmt.Println("Starting server")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "OK"})
	})
	r.Run()
}

func doSync() {
	var projects []entity.Project
	if err := dao.Collection("project").Find(nil).All(&projects); err != nil {
		panic(err)
	}
	s := service.ProjectService{}
	for _, p := range projects {
		if p.Id.Hex() == "5c70a0eca88fb51da4b59611" {
			continue
		}
		if err := s.SyncProject(p.Id.Hex(), p.Owner.Hex()); err != nil {
			log.Println("Fail to sync project", p.Id.Hex())
			panic(err)
		}
	}
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
