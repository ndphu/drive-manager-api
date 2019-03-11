package service

import (
	"encoding/base64"
	"encoding/json"
	"github.com/globalsign/mgo/bson"
	"github.com/globalsign/mgo/txn"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/google-api-helper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
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

func (s *ProjectService) CreateProject(displayName string, key []byte, owner bson.ObjectId) (*entity.Project, error) {
	kd := KeyDetails{}
	if err := json.Unmarshal(key, &kd); err != nil {
		return nil, err
	}

	if err := enableRequiredAPIs(key); err != nil {
		return nil, err
	}

	project, account, err := s.insertProject(displayName, key, owner)
	if err != nil {
		return nil, err
	}

	s.ProvisionProject(project, account)

	return project, nil
}

func (s *ProjectService) insertProject(displayName string, key []byte, owner bson.ObjectId) (*entity.Project, *entity.DriveAccount, error) {
	kd := KeyDetails{}
	if err := json.Unmarshal(key, &kd); err != nil {
		return nil, nil, err
	}
	projectUID := bson.NewObjectId()
	accountId := bson.NewObjectId()
	accountName := "admin-" + strconv.FormatInt(time.Now().Unix(), 16)
	prj := entity.Project{
		Id:          projectUID,
		Owner:       owner,
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
		Id:     projectUID,
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
	client := config.Client(oauth2.NoContext)
	srv, err := serviceusage.New(client)

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

func (s *ProjectService) ProvisionProject(project *entity.Project, adminAccount *entity.DriveAccount) {
	accSrv, _ := GetAccountService()

	config, err := google.JWTConfigFromJSON([]byte(adminAccount.Key), iam.CloudPlatformScope)
	if err != nil {
		// TODO: write provision_event FAIL TO INITIALIZE MANAGER KEY
		return
	}

	client := config.Client(oauth2.NoContext)
	iamSrv, err := iam.New(client)

	jobs := make(chan int, 100)

	for w := 1; w <= 4; w++ {
		go worker(w, jobs, iamSrv, project, accSrv)
	}

	for i := 0; i < 99; i++ {
		jobs <- i
	}
}

func worker(id int, jobs <-chan int, iamSrv *iam.Service, project *entity.Project, accSrv *AccountService) {
	log.Println("started worker", id)
	for i := range jobs {
		log.Println("creating service account", i)
		createServiceAccountAutomate(iamSrv, project, accSrv)
	}
}

func createServiceAccountAutomate(iamSrv *iam.Service, project *entity.Project, accSrv *AccountService) error {
	accountName := "account-" + strconv.FormatInt(time.Now().Unix()+int64(rand.Intn(100000)), 16)
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
			log.Println("Account may not ready at this moment. Tried:", tried)
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
