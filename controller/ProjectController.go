package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"drive-manager-api/dao"
	"drive-manager-api/entity"
	"drive-manager-api/middleware"
	"drive-manager-api/service"
	"io/ioutil"
	"log"
)

type ProjectCreateRequest struct {
	DisplayName string `json:"displayName"`
	Key         string `json:"key"`
}

func ProjectController(r *gin.RouterGroup) {
	projectService := service.GetProjectService()
	r.Use(middleware.FirebaseAuthMiddleware())

	r.GET("", func(c *gin.Context) {
		user := CurrentUser(c)
		projects := make([]entity.Project, 0)
		dao.Collection("project").Find(bson.M{
			"owner": user.Id,
		}).Select(bson.M{
			"adminKey": 0,
		}).All(&projects)
		c.JSON(200, projects)
	})

	r.GET("/:id", func(c *gin.Context) {
		user := CurrentUser(c)
		project := entity.Project{}
		if err := dao.Collection("project").Find(bson.M{
			"_id":   bson.ObjectIdHex(c.Param("id")),
			"owner": user.Id,
		}).Select(bson.M{
			"adminKey": 0,
		}).One(&project); err != nil {
			ServerError("account not found", err, c)
		} else {
			c.JSON(200, project)
		}
	})

	r.GET("/:id/accounts", func(c *gin.Context) {
		user := CurrentUser(c)
		log.Println("user", user.Id.Hex())
		log.Println("project", c.Param("id"))
		accounts := make([]entity.DriveAccount, 0)
		if err := dao.Collection("drive_account").Find(bson.M{
			"projectId": bson.ObjectIdHex(c.Param("id")),
			"owner":     user.Id,
		}).Select(bson.M{
			"key": 0,
		}).All(&accounts); err != nil {
			ServerError("account not found", err, c)
		} else {
			c.JSON(200, accounts)
		}
	})

	r.POST("", func(c *gin.Context) {
		user := CurrentUser(c)
		displayName := c.Request.FormValue("displayName")
		uploadFile, _, err := c.Request.FormFile("file")
		if err != nil {
			BadRequest("Bad Request", err, c)
			return
		}

		key, err := ioutil.ReadAll(uploadFile)
		if err != nil {
			BadRequest("Bad Request", err, c)
			return
		}

		kd := service.KeyDetails{}
		if err := json.Unmarshal(key, &kd); err != nil {
			BadRequest("fail to account key from base64", err, c)
			return
		}

		project, err := projectService.CreateProject(displayName, key, user.Id)
		if err != nil {
			ServerError("Fail to create project", err, c)
			return
		}

		c.JSON(200, project)
	})
}