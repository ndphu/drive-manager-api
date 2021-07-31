package service

import (
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/helper"
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
	"log"
	"os"
	"strconv"
	"time"
)

const DefaultDriveFileFormat = "https://www.googleapis.com/drive/v3/files/%s?alt=media&prettyPrint=false"

type GoogleService struct {

}

func (g *GoogleService) GetDownloadLink(accountId, fileId string) (*helper.DownloadDetails, error) {
	var acc entity.DriveAccount
	if err := dao.Collection("drive_account").FindId(bson.ObjectIdHex(accountId)).One(&acc); err != nil {
		log.Println("Fail to file drive account by error", err.Error())
		return nil, err
	}
	s, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		log.Println("Fail to get drive service from account key", err.Error())
		return nil, err
	}

	f, err := s.Service.Files.
		Get(fileId).
		Fields("id, name, size, mimeType").
		Do()
	if err != nil {
		log.Println("Fail to get file info from google", err.Error())
		return nil, err
	}

	var linkFormat = os.Getenv("DRIVE_FILE_DOWNLOAD_LINK_TEMPLATE")
	if linkFormat == "" {
		linkFormat = DefaultDriveFileFormat
	}

	token, err := s.GetAccessToken()
	if err != nil {
		log.Println("Fail to get access token")
		return nil, err
	}
	return &helper.DownloadDetails{
		Link:       fmt.Sprintf(linkFormat, fileId),
		Token:      token,
		File: f,
	}, nil
}

func (g *GoogleService) CreateServiceAccount(userId string, projectId string) error {
	owner := bson.ObjectIdHex(userId)
	//pid := bson.ObjectIdHex(projectId)
	adminAccount := entity.ServiceAccountAdmin{}
	err := dao.Collection("service_account_admin").Find(bson.M{"userId": owner}).One(&adminAccount)
	if err != nil {
		return err
	}
	b := []byte(adminAccount.Key)
	config, err := google.JWTConfigFromJSON(b, iam.CloudPlatformScope)
	if err != nil {
		return err
	}
	client := config.Client(oauth2.NoContext)
	srv, err := iam.New(client)

	kd := KeyDetails{}
	err = json.Unmarshal(b, &kd)
	if err != nil {
		log.Println("Fail to parse key")
		return err
	}

	accountName := "account-" + strconv.FormatInt(time.Now().Unix(), 16)
	account, err := createServiceAccount(srv, kd.ProjectId, accountName, "automate account "+accountName)
	if err != nil {
		log.Println("Fail to create service account", err)
		return nil
	}

	serviceAccountKey, err := createKeyFile(srv, account)
	if err != nil {
		log.Println("Fail to create service account key", err)
		return nil
	}
	newAccount := entity.DriveAccount{}
	if err := accountService.InitializeKey(&newAccount, serviceAccountKey); err != nil {
		log.Println("Fail to initialize account service key", err.Error())
		return err
	}
	newAccount.Name = accountName
	newAccount.Owner = owner
	if err := accountService.Save(&newAccount); err != nil {
		log.Println("Fail to persist service account", err.Error())
		return nil
	}
	if err := accountService.UpdateAccountQuotaByOwner(owner); err != nil {
		return err
	}
	log.Println("Successfully create service account")
	return nil
}
