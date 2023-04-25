package controller

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/middleware"
	"github.com/ndphu/drive-manager-api/service"
	"google.golang.org/api/iam/v1"
)

type RegisterInfo struct {
	DisplayName string `json:"displayName"`
	UserEmail   string `json:"email"`
	Password    string `json:"password"`
}

type LoginWithFirebase struct {
	Token string `json:"token"`
}

type UserInfo struct {
	Id            primitive.ObjectID `json:"id" bson:"_id"`
	Email         string        `json:"email" bson:"email"`
	Roles         []string      `json:"roles" bson:"roles"`
	DisplayName   string        `json:"displayName" bson:"displayName"`
	NoOfAdminKeys int           `json:"noOfAdminKeys" bson:"noOfAdminKeys"`
	NoOfAccounts  int           `json:"noOfAccounts" bson:"noOfAccounts"`
}

func UserController(r *gin.RouterGroup) {
	authService, _ := service.GetAuthService()
	//accountService := service.GetAccountService()

	r.POST("/register", func(c *gin.Context) {
		ri := RegisterInfo{}
		err := c.ShouldBindJSON(&ri)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"success": false, "error": err.Error()})
		}
		user, err := authService.CreateUserWithEmail(ri.UserEmail, ri.Password, ri.DisplayName)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err, "message": err.Error()})
			return
		}

		c.JSON(200, gin.H{"user": user})
	})

	r.POST("/login/firebase", func(c *gin.Context) {
		loginInfo := LoginWithFirebase{}
		err := c.ShouldBindJSON(&loginInfo)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": err})
			return
		}
		user, jwtToken, err := authService.LoginWithFirebaseToken(loginInfo.Token)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": err})
			return
		}

		c.JSON(200, gin.H{"user": user, "jwtToken": jwtToken})
	})

	info := r.Group("/manage").Use(middleware.FirebaseAuthMiddleware())
	{
		info.GET("/info", func(c *gin.Context) {
			val, _ := c.Get("user")
			user := val.(*entity.User)

			//userInfo := make([]UserInfo, 0)
			userInfo := make([]interface{}, 0)
			dao.User().Pipe([]bson.M{
				{"$match": bson.M{"_id": user.Id}},
				{"$lookup": bson.M{
					"from":         "service_account_admin",
					"localField":   "_id",
					"foreignField": "userId",
					"as":           "adminKeys",
				}},
				{"$lookup": bson.M{
					"from":         "drive_account",
					"localField":   "_id",
					"foreignField": "owner",
					"as":           "accounts",
				}},
				{"$lookup": bson.M{
					"from":         "service_token",
					"localField":   "_id",
					"foreignField": "userId",
					"as":           "serviceTokens",
				}},
				{"$project": bson.M{
					"_id":           1,
					"email":         1,
					"displayName":   1,
					"roles":         1,
					"serviceTokens": 1,
					"noOfAdminKeys": bson.M{"$size": "$adminKeys"},
					"noOfAccounts":  bson.M{"$size": "$accounts"},
				}},
			}, &userInfo)

			c.JSON(200, userInfo[0])
		})

		//info.POST("/adminKey", func(c *gin.Context) {
		//	body, err := c.GetRawData()
		//	if err != nil {
		//		BadRequest("Request required body as base64", err, c)
		//		return
		//	}
		//	keyDecoded, err := base64.StdEncoding.DecodeString(string(body))
		//	if err != nil {
		//		BadRequest("Fail to decode base64 key data", err, c)
		//		return
		//	}
		//	val, _ := c.Get("user")
		//	user := val.(*entity.User)
		//	adminAccount := entity.ServiceAccountAdmin{}
		//	err = dao.ServiceAccountAdmin().FindOne(bson.M{"userId": user.Id}, &adminAccount)
		//	if err != nil {
		//		adminAccount = entity.ServiceAccountAdmin{
		//			Id:     bson.NewObjectId(),
		//			UserId: user.Id,
		//			Key:    string(keyDecoded),
		//		}
		//		dao.ServiceAccountAdmin().Insert(&adminAccount)
		//		c.JSON(200, adminAccount)
		//	} else {
		//		adminAccount.Key = string(keyDecoded)
		//		dao.Collection("service_account_admin").UpdateId(adminAccount.Id, &adminAccount)
		//		c.JSON(200, adminAccount)
		//	}
		//})

		//info.POST("/increaseStorage", func(c *gin.Context) {
		//	user := CurrentUser(c)
		//	adminAccount := entity.ServiceAccountAdmin{}
		//	err := dao.Collection("service_account_admin").Find(bson.M{"userId": user.Id}).One(&adminAccount)
		//	if err != nil {
		//		ServerError("no admin account available", err, c)
		//		return
		//	}
		//	b := []byte(adminAccount.Key)
		//	config, err := google.JWTConfigFromJSON(b, iam.CloudPlatformScope)
		//	if err != nil {
		//		ServerError("Unable to parse client secret file to config: %v", err, c)
		//		return
		//	}
		//	client := config.Client(oauth2.NoContext)
		//	srv, err := iam.New(client)
		//
		//	kd := service.KeyDetails{}
		//	json.Unmarshal(b, &kd)
		//
		//	accountName := "account-" + strconv.FormatInt(time.Now().Unix(), 16)
		//	account, err := createServiceAccount(srv, kd.ProjectId, accountName, "automate account "+accountName)
		//	if err != nil {
		//		ServerError("fail to create service account", err, c)
		//		return
		//	}
		//
		//	serviceAccountKey, err := createKeyFile(srv, account)
		//	if err != nil {
		//		ServerError("fail to create service account key", err, c)
		//		return
		//	}
		//	newAccount := entity.DriveAccount{}
		//
		//	accountService.InitializeKey(&newAccount, serviceAccountKey)
		//	newAccount.Name = accountName
		//	newAccount.Owner = user.Id
		//	if err := accountService.Save(&newAccount); err != nil {
		//		ServerError("fail to persist service account", err, c)
		//		return
		//	}
		//
		//	accountService.UpdateAccountQuotaByOwner(user.Id)
		//
		//	c.JSON(200, account)
		//})

		info.POST("/createServiceToken", func(c *gin.Context) {
			serviceToken, err := authService.NewServiceToken(CurrentUser(c))
			if err != nil {
				c.AbortWithStatusJSON(500, gin.H{"success": false, "error": err.Error()})
				return
			}
			c.JSON(200, serviceToken)
		})
	}

}

func createServiceAccount(s *iam.Service, projectId string, name string, displayName string) (*iam.ServiceAccount, error) {
	req := iam.CreateServiceAccountRequest{}
	req.AccountId = name
	req.ServiceAccount = &iam.ServiceAccount{
		DisplayName: displayName,
	}
	account, err := s.Projects.ServiceAccounts.Create("projects/"+projectId, &req).Do()
	if err != nil {
		return nil, err
	}
	return account, nil
}

func createKeyFile(srv *iam.Service, account *iam.ServiceAccount) ([]byte, error) {
	key, err := srv.Projects.ServiceAccounts.Keys.Create("projects/-/serviceAccounts/"+account.Email, &iam.CreateServiceAccountKeyRequest{}).Do()
	if err != nil {
		return make([]byte, 0), err
	}
	//log.Println(key.MarshalJSON())
	keyBytes, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return make([]byte, 0), err
	}
	return keyBytes, nil
}
