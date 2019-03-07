package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/middleware"
	"github.com/ndphu/drive-manager-api/service"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/serviceusage/v1"
	"io/ioutil"
	"log"
	"strconv"
	"time"
)

type ProjectCreateRequest struct {
	DisplayName string `json:"displayName"`
	Key         string `json:"key"`
}

func ProjectController(r *gin.RouterGroup) {
	accountService, _ := service.GetAccountService()

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
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			BadRequest("Bad Request", err, c)
			return
		}

		key, err := ioutil.ReadAll(file)
		if err != nil {
			BadRequest("Bad Request", err, c)
			return
		}

		kd := service.KeyDetails{}
		if err := json.Unmarshal(key, &kd); err != nil {
			BadRequest("fail to account key from base64", err, c)
			return
		}

		config, err := google.JWTConfigFromJSON(key, serviceusage.CloudPlatformScope)
		if err != nil {
			ServerError("Google Service Error", err, c)
			return
		}
		client := config.Client(oauth2.NoContext)
		srv, err := serviceusage.New(client)

		stateResp, err := srv.Services.
			List("projects/" + kd.ProjectId).
			Filter("state:ENABLED").Do()
		if err != nil {
			ServerError("Auto Configure Failed", err, c)
			return
		}

		enabledServices := make(map[string]bool)
		for _, s := range stateResp.Services {
			enabledServices[s.Name] = true
		}

		for _, name := range []string{"iam.googleapis.com", "drive.googleapis.com"} {
			if enabledServices[name] {
				log.Println("service", name, "is already enabled")
				continue
			}
			log.Println("enabled service", name, "...")
			_, err := srv.Services.Enable("projects/"+kd.ProjectId+"/services/"+name,
				&serviceusage.EnableServiceRequest{}).Do()
			if err != nil {
				ServerError("Auto Configure Failed", err, c)
				return
			}
		}

		prj := entity.Project{
			Id:          bson.NewObjectId(),
			Owner:       user.Id,
			DisplayName: displayName,
			AdminKey:    string(key),
			ProjectId:   kd.ProjectId,
		}
		if err := dao.Collection("project").Insert(&prj); err != nil {
			ServerError("fail to insert new project", err, c)
			return
		}
		c.JSON(200, prj)
	})

	r.POST("/:id/addServiceAccount", func(c *gin.Context) {
		user := CurrentUser(c)
		project := entity.Project{}
		if err := dao.Collection("drive_account").
			Find(bson.M{
				"_id":   bson.ObjectIdHex(c.Param("id")),
				"owner": user.Id,
			}).
			One(&project); err != nil {
			ServerError("Account not found", err, c)
			return
		}

		b := []byte(project.AdminKey)

		config, err := google.JWTConfigFromJSON(b, iam.CloudPlatformScope)
		if err != nil {
			ServerError("Unable to parse client secret file to config: %v", err, c)
			return
		}
		client := config.Client(oauth2.NoContext)
		srv, err := iam.New(client)

		kd := service.KeyDetails{}
		json.Unmarshal(b, &kd)

		accountName := "account-" + strconv.FormatInt(time.Now().Unix(), 16)
		account, err := createServiceAccount(srv, kd.ProjectId, accountName, "automate account "+accountName)
		if err != nil {
			ServerError("fail to create service account", err, c)
			return
		}

		serviceAccountKey, err := createKeyFile(srv, account)
		if err != nil {
			ServerError("fail to create service account key", err, c)
			return
		}
		newAccount := entity.DriveAccount{}

		accountService.InitializeKey(&newAccount, serviceAccountKey)
		newAccount.Name = accountName
		newAccount.Owner = user.Id
		newAccount.ProjectId = project.Id
		if err := accountService.Save(&newAccount); err != nil {
			ServerError("fail to persist service account", err, c)
			return
		}

		accountService.UpdateCachedQuota(&newAccount)

		c.JSON(200, account)
	})

}
