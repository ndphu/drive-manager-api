package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/service"
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

	r.DELETE("/project/:id", func(c *gin.Context) {
		user := CurrentUser(c)
		projectId := c.Param("id")
		p, _ := s.GetProject(projectId)
		if p.Owner.Hex() != user.Id.Hex() {
			c.AbortWithStatusJSON(403, gin.H{"error": "project not found"})
			return
		}
		if err := s.DeleteProject(projectId); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"success": true})
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
		count, err := strconv.Atoi(c.Query("count"))
		if err != nil {
			count = 1
		}
		log.Println("Adding", count, "new account(s) for project", projectId)
		accounts := make([]*entity.DriveAccount, 0)
		for i := 0; i < count; i++ {
			if account, err := accountService.CreateServiceAccount(projectId, user.Id.Hex()); err != nil {
				break
			} else {
				account.Key = ""
				accounts = append(accounts, account)
			}
		}
		c.JSON(200, gin.H{
			"success":  true,
			"accounts": accounts,
		})
	})

	r.POST("/project/:id/syncQuota", func(c *gin.Context) {
		//user := CurrentUser(c)
		projectId := c.Param("id")
		// TODO check permission
		if err := s.SyncProjectQuota(projectId); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{
				"success": true,
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

	r.POST("/project/:id/sync", func(c *gin.Context) {
		user := CurrentUser(c)
		projectId := c.Param("id")
		if count, err := dao.Collection("project").Find(bson.M{
			"_id":   bson.ObjectIdHex(projectId),
			"owner": user.Id,
		}).Count(); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			if count == 0 {
				c.JSON(404, gin.H{"error": "project not found"})
			} else {
				if err := s.SyncProjectWithGoogle(projectId); err != nil {
					c.AbortWithStatusJSON(500, gin.H{"error": "fail to sync project by error" + err.Error()})
				} else {
					c.JSON(200, gin.H{"success": true})
				}
			}
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
