package service

import (
	"context"
	"drive-manager-api/dao"
	"drive-manager-api/entity"
	"drive-manager-api/helper"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/globalsign/mgo/bson"
	"github.com/globalsign/mgo/txn"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
	"log"
	"math/rand"
	"strconv"
	"time"
)

var projectService *ProjectService

type ProjectService struct{}

func GetProjectService() *ProjectService {
	if projectService == nil {
		projectService = &ProjectService{

		}
	}

	return projectService
}

func (s *ProjectService) CreateProject(displayName string, key []byte, numberOfAccounts int, owner bson.ObjectId) (*entity.Project, error) {
	kd := KeyDetails{}
	if err := json.Unmarshal(key, &kd); err != nil {
		log.Println("Fail to parse project admin key by error", err.Error())
		return nil, err
	}
	// validate project ID from key file should be unique
	if count, err := dao.Collection("project").Find(bson.M{"projectId": kd.ProjectId}).Count(); err != nil {
		log.Println("Fail to check key unique with database", err.Error())
		return nil, err
	} else if count > 0 {
		log.Println("Project ID", kd.ProjectId, "is not unique")
		return nil, errors.New("DuplicatedGoogleProjectId")
	}

	if err := enableRequiredAPIs(key); err != nil {
		log.Println("Fail to enable required API by error", err.Error())
		return nil, err
	}

	project, account, err := s.insertProject(displayName, key, owner)
	if err != nil {
		log.Println("Fail to insert project to database by error", err.Error())
		return nil, err
	}

	if numberOfAccounts > 0 {
		if err := s.ProvisionProject(project, account, numberOfAccounts); err != nil {
			log.Println("Fail to provision project by error", err.Error())
			return nil, err
		}
	}

	return project, nil
}

func (s *ProjectService) insertProject(displayName string, key []byte, owner bson.ObjectId) (*entity.Project, *entity.DriveAccount, error) {
	kd := KeyDetails{}
	if err := json.Unmarshal(key, &kd); err != nil {
		return nil, nil, err
	}
	pid := bson.NewObjectId()
	accountId := bson.NewObjectId()
	prj := entity.Project{
		Id:          pid,
		DisplayName: displayName,
		ProjectId:   kd.ProjectId,
		Owner:       owner,
	}

	acc := entity.DriveAccount{
		Id:          accountId,
		ProjectId:   pid,
		Name:        "admin-account",
		Key:         string(key),
		Desc:        "Admin Account",
		Type:        "service_account_admin",
		ClientEmail: kd.ClientEmail,
		ClientId:    kd.ClientId,
		Owner:       owner,
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
		Id:     pid,
		Assert: txn.DocMissing,
		Insert: prj,
	}}
	if err := runner.Run(ops, "", nil); err != nil {
		return nil, nil, err
	}
	return &prj, &acc, nil
}

func enableRequiredAPIs(key []byte) error {
	kd := KeyDetails{}
	if err := json.Unmarshal(key, &kd); err != nil {
		return err
	}
	config, err := google.JWTConfigFromJSON(key, serviceusage.CloudPlatformScope)
	if err != nil {
		return err
	}
	ctx := context.Background()
	//client := config.Client(ctx)
	srv, err := serviceusage.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))

	stateResp, err := srv.Services.
		List("projects/" + kd.ProjectId).
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
		_, err := srv.Services.Enable("projects/"+kd.ProjectId+"/services/"+name,
			&serviceusage.EnableServiceRequest{}).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ProjectService) ProvisionProject(project *entity.Project, adminAccount *entity.DriveAccount, numberOfAccounts int) error {
	accSrv := GetAccountService()

	config, err := google.JWTConfigFromJSON([]byte(adminAccount.Key), iam.CloudPlatformScope)
	if err != nil {
		log.Println("Fail to initialize with project admin key", err.Error())
		return nil
	}

	iamService, err := iam.NewService(context.Background(), option.WithTokenSource(config.TokenSource(context.Background())))
	if err != nil {
		log.Println("Fail to create IAM service instance", err.Error())
		return err
	}

	for i := 0; i < numberOfAccounts; i++ {
		if err := createServiceAccountAutomate(iamService, project, accSrv); err != nil {
			log.Println("Fail to create service account by error", err.Error())
			return err
		}
	}

	return nil

	//
	//jobs := make(chan int, numberOfAccounts)
	//
	//for w := 1; w <= 4; w++ {
	//	go worker(w, jobs, iamSrv, project, accSrv)
	//}
	//
	//for i := 0; i < numberOfAccounts; i++ {
	//	jobs <- i
	//}
}

func worker(id int, jobs <-chan int, iamSrv *iam.Service, project *entity.Project, accSrv *AccountService) {
	log.Println("started worker", id)
	for i := range jobs {
		log.Println("creating service account", i)
		createServiceAccountAutomate(iamSrv, project, accSrv)
	}
}

func createServiceAccountAutomate(iamSrv *iam.Service, project *entity.Project, accSrv *AccountService) error {
	accountName := "sa-" + strconv.FormatInt(time.Now().Unix()+int64(rand.Intn(100000)), 16)
	account, err := createServiceAccount(iamSrv, project.ProjectId, accountName, "automate account "+accountName)
	if err != nil {
		// TODO: write provision_event FAIL TO CREATE SERVICE ACCOUNT
		log.Println("Fail to create service account by error", err.Error())
		return err
	}
	serviceAccountKey, err := createKeyFile(iamSrv, account)
	if err != nil {
		// TODO: write provision_event FAIL TO GENERATE KEY FILE
		log.Println("Fail to generate service account key file by error", err.Error())
		return err
	}
	newAcc := entity.DriveAccount{}
	if err := accSrv.InitializeKey(&newAcc, serviceAccountKey); err != nil {
		// TODO: write provision_event FAIL TO PARSE GENERATED KEY FILE
		log.Println("Fail to parse service account key file by error", err.Error())
		return err
	}
	newAcc.Name = accountName
	newAcc.Owner = project.Owner
	newAcc.ProjectId = project.Id
	newAcc.Key = string(serviceAccountKey)

	srv, err := helper.GetDriveService(serviceAccountKey)
	if err != nil {
		log.Println("Fail to get drive service from account key by error", err.Error())
		return err
	}
	tries := 0
	for {
		tries++
		quota, err := srv.GetQuotaUsage()
		if err != nil {
			log.Println("Account may not ready at this moment. Tries:", tries)
			time.Sleep(5 * time.Second)
		} else {
			log.Println("Account is now available")
			newAcc.Usage = quota.Usage
			newAcc.Limit = quota.Limit
			newAcc.QuotaUpdateTimestamp = time.Now()
			break
		}
		if tries >= 30 {
			return err
		}
	}

	if err := accSrv.Save(&newAcc); err != nil {
		// TODO: write provision_event FAIL TO SAVE NEW ACCOUNT TO DB
		log.Println("Fail to save new service account to DB")
		return err
	}

	return nil
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
	keyBytes, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return make([]byte, 0), err
	}
	return keyBytes, nil
}

func parseKeyDetails(key []byte) (*KeyDetails, error) {
	kd := KeyDetails{}
	if err := json.Unmarshal(key, &kd); err != nil {
		log.Println("Fail to parse project admin key by error", err.Error())
		return nil, err
	}
	return &kd, nil
}
