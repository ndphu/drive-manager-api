package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/globalsign/mgo/txn"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/middleware"
	"github.com/ndphu/drive-manager-api/service"
	"github.com/ndphu/google-api-helper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/serviceusage/v1"
	"io/ioutil"
	"log"
	"math/rand"
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

		if err := enableRequiredAPIs(kd.ProjectId, key); err != nil {
			ServerError("fail to enable required APIs", err, c)
			return
		}

		prj, acc, err := insertProject(user, displayName, kd, key)
		if err != nil {
			ServerError("fail to create project", err, c)
			return
		}

		go provisionProject(prj, acc)

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

		var b []byte

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

func insertProject(user *entity.User, displayName string, kd service.KeyDetails, key []byte) (*entity.Project, *entity.DriveAccount, error) {
	projectUID := bson.NewObjectId()
	accountId := bson.NewObjectId()
	accountName := "admin-" + strconv.FormatInt(time.Now().Unix(), 16)
	prj := entity.Project{
		Id:          projectUID,
		Owner:       user.Id,
		DisplayName: displayName,
		ProjectId:   kd.ProjectId,
	}

	acc := entity.DriveAccount{
		Id:          accountId,
		ProjectId:   projectUID,
		Name:        accountName,
		Key:         string(key),
		Desc:        "Admin Account",
		Type:        "service_account_admin",
		ClientEmail: kd.ClientEmail,
		ClientId:    kd.ClientId,
		Owner:       user.Id,
	}

	tc := dao.GetDB().C("transaction")
	runner := txn.NewRunner(tc)
	ops := []txn.Op{{
		C:      "drive_account",
		Id:     accountId,
		Assert: txn.DocMissing,
		Insert: acc,
	}, {
		C:      "project",
		Id:     projectUID,
		Assert: txn.DocMissing,
		Insert: prj,
	}}
	if err := runner.Run(ops, "", nil); err != nil {
		return nil, nil, err
	}
	return &prj, &acc, nil
}

func enableRequiredAPIs(projectId string, key []byte) error {
	config, err := google.JWTConfigFromJSON(key, serviceusage.CloudPlatformScope)
	if err != nil {
		return err
	}
	client := config.Client(oauth2.NoContext)
	srv, err := serviceusage.New(client)

	stateResp, err := srv.Services.
		List("projects/" + projectId).
		Filter("state:ENABLED").Do()
	if err != nil {
		return err
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
		_, err := srv.Services.Enable("projects/"+projectId+"/services/"+name,
			&serviceusage.EnableServiceRequest{}).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

func provisionProject(project *entity.Project, adminAccount *entity.DriveAccount) {
	accSrv, _ := service.GetAccountService()

	config, err := google.JWTConfigFromJSON([]byte(adminAccount.Key), iam.CloudPlatformScope)
	if err != nil {
		// TODO: write provision_event FAIL TO INITIALIZE MANAGER KEY
		return
	}

	client := config.Client(oauth2.NoContext)
	iamSrv, err := iam.New(client)

	jobs := make(chan int, 100)

	for w:=1; w <= 4; w++ {
		go worker(w, jobs, iamSrv, project, accSrv)
	}

	for i := 0; i < 99; i++ {
		jobs <- i
	}
}

func worker(id int, jobs <- chan int, iamSrv *iam.Service, project *entity.Project, accSrv *service.AccountService)  {
	log.Println("started worker", id)
	for i := range jobs {
		log.Println("creating service account", i)
		createServiceAccountAutomate(iamSrv, project, accSrv)
	}
}

func createServiceAccountAutomate(iamSrv *iam.Service, project *entity.Project, accSrv *service.AccountService) error {
	accountName := "account-" + strconv.FormatInt(time.Now().Unix() + int64(rand.Intn(100000)), 16)
	account, err := createServiceAccount(iamSrv, project.ProjectId, accountName, "automate account "+accountName)
	if err != nil {
		// TODO: write provision_event FAIL TO CREATE SERVICE ACCOUNT
		return err
	}
	serviceAccountKey, err := createKeyFile(iamSrv, account)
	if err != nil {
		// TODO: write provision_event FAIL TO GENERATE KEY FILE
		return err
	}
	newAcc := entity.DriveAccount{}
	if err := accSrv.InitializeKey(&newAcc, serviceAccountKey); err != nil {
		// TODO: write provision_event FAIL TO PARSE GENERATED KEY FILE
		return err
	}
	newAcc.Name = accountName
	newAcc.Owner = project.Owner
	newAcc.ProjectId = project.Id
	newAcc.Key = string(serviceAccountKey)

	srv, err := google_api_helper.GetDriveService(serviceAccountKey)

	tried := 0
	for {
		tried ++
		quota, err := srv.GetQuotaUsage()

		if err != nil {
			log.Println("Account may not ready at this moment. Tried:" , tried)
			time.Sleep(5 * time.Second)
		} else {
			log.Println("Account is now available")
			newAcc.Usage = quota.Usage
			newAcc.Limit = quota.Limit
			newAcc.QuotaUpdateTimestamp = time.Now()
			break
		}

		if tried >= 30 {
			return err
		}
	}

	if err := accSrv.Save(&newAcc); err != nil {
		// TODO: write provision_event FAIL TO SAVE NEW ACCOUNT TO DB
		return err
	}

	return nil
}
