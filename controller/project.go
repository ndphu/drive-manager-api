package controller

import (
	"drive-manager-api/dao"
	"drive-manager-api/entity"
	"drive-manager-api/middleware"
	"drive-manager-api/service"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type ProjectCreateRequest struct {
	DisplayName string `json:"displayName"`
	Key         string `json:"key"`
}

type ProjectLookup struct {
	Id               bson.ObjectId         `json:"id" bson:"_id"`
	DisplayName      string                `json:"displayName" bson:"displayName"`
	ProjectId        string                `json:"projectId" bson:"projectId"`
	Owner            bson.ObjectId         `json:"owner" bson:"owner"`
	Accounts         []entity.DriveAccount `json:"accounts" bson:"accounts"`
	NumberOfAccounts int                   `json:"numberOfAccounts" bson:"numberOfAccounts"`
}

func ProjectController(r *gin.RouterGroup) {
	s := service.GetProjectService()
	accountService := service.GetAccountService()
	r.Use(middleware.FirebaseAuthMiddleware())

	r.GET("/projects", func(c *gin.Context) {
		user := CurrentUser(c)
		projects := make([]ProjectLookup, 0)
		if err := dao.Collection("project").Pipe([]bson.M{
			{
				"$match": bson.M{"owner": user.Id},
			},
			{
				"$lookup": bson.M{
					"from":         "drive_account",
					"localField":   "_id",
					"foreignField": "projectId",
					"as":           "accounts",
				},
			},
			{
				"$project": bson.M{
					"id":          1,
					"displayName": 1,
					"owner":       1,
					"projectId":   1,
					"numberOfAccounts": bson.M{
						"$size": "$accounts",
					},
				},
			},
		}).All(&projects); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"projects": projects})
		}
	})

	r.GET("/project/:id", func(c *gin.Context) {
		user := CurrentUser(c)
		projectId := c.Param("id")
		if project, err := queryProjectLookup(user.Id.Hex(), projectId); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{
				"project": project,
			})
		}
	})

	r.GET("/project/:id/accounts", func(c *gin.Context) {
		user := CurrentUser(c)
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

	r.POST("/project/:id/newAccount", func(c *gin.Context) {
		user := CurrentUser(c)
		projectId := c.Param("id")
		log.Println("Adding new account(s) for project", projectId)
		account, err := accountService.CreateServiceAccount(projectId, user.Id.Hex())
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			account.Key = ""
			c.JSON(200, gin.H{
				"success": true,
				"account": account,
			})
		}
	})

	r.POST("/projects", func(c *gin.Context) {
		user := CurrentUser(c)
		displayName := strings.TrimSpace(c.Request.FormValue("displayName"))
		if displayName == "" {
			c.AbortWithStatusJSON(400, gin.H{"error": "Project name could not be empty"})
			return
		}
		numberOfAccounts := 0
		num := c.Request.FormValue("numberOfAccounts")
		if num != "" {
			parsed, err := strconv.Atoi(num)
			if err != nil {
				c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
				return
			} else {
				numberOfAccounts = parsed
			}
		}

		uploadFile, _, err := c.Request.FormFile("file")
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
			return
		}
		key, err := ioutil.ReadAll(uploadFile)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
			return
		}

		project, err := s.CreateProject(displayName, key, numberOfAccounts, user.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		p, err := queryProjectLookup(user.Id.Hex(), project.Id.Hex())
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"success": true, "project": p})
		}

	})
}

func queryProjectLookup(userId, projectId string) (*ProjectLookup, error) {
	var project ProjectLookup
	if err := dao.Collection("project").Pipe([]bson.M{
		{
			"$match": bson.M{
				"$and": []bson.M{
					{"_id": bson.ObjectIdHex(projectId)},
					{"owner": bson.ObjectIdHex(userId)},
				},
			},
		},
		{
			"$lookup": bson.M{
				"from": "drive_account",
				"let":  bson.M{"projectId": "$_id"},
				"pipeline": []bson.M{
					{"$match": bson.M{
						"$expr": bson.M{
							"$eq": []string{"$projectId", "$$projectId"},
						},
					}},
					{"$project": bson.M{
						"key": 0,
					}},
				},
				"as": "accounts",
			},
		},
	}).One(&project); err != nil {
		return nil, err
	}
	return &project, nil
}
