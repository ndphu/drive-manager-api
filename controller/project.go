package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
	Id               primitive.ObjectID    `json:"id" bson:"_id"`
	DisplayName      string                `json:"displayName" bson:"displayName"`
	ProjectId        string                `json:"projectId" bson:"projectId"`
	Owner            primitive.ObjectID    `json:"owner" bson:"owner"`
	Accounts         []entity.DriveAccount `json:"accounts" bson:"accounts"`
	Disabled         bool                  `json:"disabled" bson:"disabled"`
	NumberOfAccounts int                   `json:"numberOfAccounts" bson:"numberOfAccounts"`
}

func ProjectController(r *gin.RouterGroup) {
	s := service.GetProjectService()
	accountService := service.GetAccountService()

	r.GET("/projects", func(c *gin.Context) {
		user := CurrentUser(c)
		//includeDisabled := c.Query("includeDisabled") == "true"
		//match := bson.M{
		//	"owner": user.Id,
		//}
		//if !includeDisabled {
		//	match["disabled"] = bson.M{"$ne": true}
		//}
		projects := make([]ProjectLookup, 0)

		matchStage := bson.D{{"$match", bson.D{{"owner", user.Id}}}}
		lookupStage := bson.D{{"$lookup", bson.D{
			{"from", "drive_account"},
			{"localField", "_id"},
			{"foreignField", "projectId"},
			{"as", "accounts"},
		}}}
		sortStage := bson.D{{"$sort", bson.D{{"disabled", 1}, {"_id", 1}}}}
		projectStage := bson.D{{"$project", bson.D{
			{"id", 1},
			{"displayName", 1},
			{"owner", 1},
			{"disabled", 1},
			{"projectId", 1},
			{"numberOfAccounts", bson.D{
				{"$size", "$accounts"},
			}},
		}}}

		aggregate, err := dao.RawCollection("project").Aggregate(context.Background(), mongo.Pipeline{matchStage, lookupStage, sortStage, projectStage})
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := aggregate.All(context.Background(), &projects); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"projects": projects})
		}

		//if err := dao.Project().Pipe([]bson.M{
		//	{
		//		"$match": match,
		//	},
		//	{
		//		"$lookup": bson.M{
		//			"from":         "drive_account",
		//			"localField":   "_id",
		//			"foreignField": "projectId",
		//			"as":           "accounts",
		//		},
		//	},
		//	{
		//		"$sort": bson.M{
		//			"disabled": 1,
		//			"_id":      1,
		//		},
		//	},
		//	{
		//		"$project": bson.M{
		//			"id":          1,
		//			"displayName": 1,
		//			"owner":       1,
		//			"disabled":    1,
		//			"projectId":   1,
		//			"numberOfAccounts": bson.M{
		//				"$size": "$accounts",
		//			},
		//		},
		//	},
		//}, &projects); err != nil {
		//	c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		//} else {
		//	c.JSON(200, gin.H{"projects": projects})
		//}
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
		if err := dao.DriveAccount().Template(func(col *mongo.Collection) error {
			//return col.Find(context.Background(), bson.M{
			//	"projectId": primitive.ObjectIDFromHex(c.Param("id")),
			//	"owner":     user.Id,
			//}).Select(bson.M{
			//	"key": 0,
			//}).All(&accounts)
			projectIdHex, _ := primitive.ObjectIDFromHex(c.Param("id"))
			if cur, err := col.Find(context.Background(), bson.M{
				"projectId": projectIdHex,
				"owner":     user.Id,
			}); err != nil {
				return err
			} else {
				return cur.All(context.Background(), accounts)
			}
		}); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
			return
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

	r.PUT("/project/:id/field/:field/value/:value", func(c *gin.Context) {
		projectId := c.Param("id")
		field := c.Param("field")
		switch field {
		case "disabled":
			{
				var err error
				if c.Param("value") == "true" {
					err = s.DisableProject(projectId)
				} else {
					err = s.EnableProject(projectId)
				}
				if err != nil {
					c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
					return
				} else {
					c.JSON(200, gin.H{"success": true})
				}
				break
			}
		default:
			log.Println("Unknown field", field)
			c.AbortWithStatusJSON(400, gin.H{"success": false, "error": "unknown field: " + field})
		}
		//if len(set) > 0 {
		//	if err := dao.Project().UpdateId(primitive.ObjectIDFromHex(c.Param("id")), bson.M{
		//		"$set": set,
		//	}); err != nil {
		//		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		//		return
		//	} else {
		//		c.JSON(200, gin.H{"success": true})
		//	}
		//} else {
		//	c.AbortWithStatusJSON(400, gin.H{"success": false, "error": "Unknown field:" + field})
		//	return
		//}
	})

	r.POST("/project/:id/sync", func(c *gin.Context) {
		user := CurrentUser(c)
		projectId := c.Param("id")
		projectIdHex, _ := primitive.ObjectIDFromHex(projectId)
		if count, err := dao.RawCollection("project").CountDocuments(context.Background(), bson.D{
			{"_id", projectIdHex},
			{"owner", user.Id}}); err != nil {
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
	projectIdHex, _ := primitive.ObjectIDFromHex(projectId)
	userIdHex, _ := primitive.ObjectIDFromHex(userId)
	matchStage := bson.D{{"$match", bson.D{
		{"$and", bson.D{
			{"_id", projectIdHex},
			{"owner", userIdHex},
		},
		},
	}}}
	lookupStage := bson.D{
		{"$lookup", bson.D{
			{"from", "drive_account"},
			{"let", bson.D{{"projectId", "$_id"}}},
			{"pipeline", bson.D{
				{"$match", bson.D{
					{"$expr", bson.D{
						{"$eq", []string{"$projectId", "$$projectId"}},
					}},
				}},
				{"$project", bson.D{{"key", 0}}},
			}},
			{"as", "accounts"},
		}},
	}
	cursor, err := dao.RawCollection("project").Aggregate(context.Background(), mongo.Pipeline{matchStage, lookupStage})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(context.Background(), &project); err != nil {
		return nil, err
	} else {
		return &project, err
	}
	//if err := dao.Project().PipeOne([]bson.M{
	//	{
	//		"$match": bson.M{
	//			"$and": []bson.M{
	//				{"_id": primitive.ObjectIDFromHex(projectId)},
	//				{"owner": primitive.ObjectIDFromHex(userId)},
	//			},
	//		},
	//	},
	//	{
	//		"$lookup": bson.M{
	//			"from": "drive_account",
	//			"let":  bson.M{"projectId": "$_id"},
	//			"pipeline": []bson.M{
	//				{"$match": bson.M{
	//					"$expr": bson.M{
	//						"$eq": []string{"$projectId", "$$projectId"},
	//					},
	//				}},
	//				{"$project": bson.M{
	//					"key": 0,
	//				}},
	//			},
	//			"as": "accounts",
	//		},
	//	},
	//}, &project); err != nil {
	//	return nil, err
	//}
	//return &project, nil
}
