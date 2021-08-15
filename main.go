package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/controller"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/middleware"
)

func main() {
	err := dao.Init()
	if err != nil {
		panic(err)
	}
	defer dao.Close()

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
