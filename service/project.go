package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/helper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

func (s *ProjectService) CreateProject(displayName string, key []byte, numberOfAccounts int, owner primitive.ObjectID) (*entity.Project, error) {
	kd := KeyDetails{}
	if err := json.Unmarshal(key, &kd); err != nil {
		log.Println("Fail to parse project admin key by error", err.Error())
		return nil, err
	}
	// validate project ID from key file should be unique
	if count, err := dao.Project().CountDocuments(context.Background(), bson.D{{"projectId", kd.ProjectId}}); err != nil {
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

func (s *ProjectService) insertProject(displayName string, key []byte, owner primitive.ObjectID) (*entity.Project, *entity.DriveAccount, error) {
	kd := KeyDetails{}
	if err := json.Unmarshal(key, &kd); err != nil {
		return nil, nil, err
	}
	pid := primitive.NewObjectID()
	accountId := primitive.NewObjectID()
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

	if transaction, err := dao.ExecTransaction(func(sessCtx mongo.SessionContext) (interface{}, error) {
		log.Println("Inserting project", prj.Id.Hex())
		result, err := dao.Project().InsertOne(context.Background(), prj)
		if err != nil {
			log.Println("Fail to insert project", prj.Id.Hex())
			return nil, err
		}
		log.Println("Insert project", prj.Id.Hex(), prj.DisplayName, "successfully")
		log.Println("Inserting admin account", acc.Id.Hex(), "for project", prj.Id.Hex())
		result, err = dao.DriveAccount().InsertOne(context.Background(), acc)
		if err != nil {
			log.Println("Fail to insert admin account", acc.Id.Hex(), "for project", prj.Id.Hex())
			return nil, err
		}
		log.Println("Insert admin account", acc.Id.Hex(), acc.ClientEmail, "for project", prj.Id.Hex(), prj.DisplayName, "successfully")
		return result, nil
	}); err != nil {
		return nil, nil, err
	} else {
		log.Println("Transaction status:", transaction)
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

func (s *ProjectService) SyncProject(projectId string, userId string) error {
	var p entity.Project
	projectIdHex, _ := primitive.ObjectIDFromHex(projectId)
	userIdHex, _ := primitive.ObjectIDFromHex(userId)
	if err := dao.Project().FindOne(context.Background(), bson.D{{"_id", projectIdHex}, {"owner", userIdHex}}).Decode(&p);
		err != nil {
		log.Println("Fail to find project to perform sync by error", err.Error())
		return err
	}
	var accList []entity.DriveAccount
	if cursor, err := dao.DriveAccount().Find(context.Background(), bson.D{
		{"projectId", projectIdHex},
		{"owner", userIdHex},
	}); err != nil {
		log.Println("Fail to get account list by error", err.Error())
		return err
	} else {
		if err := cursor.All(context.Background(), &accList); err != nil {
			return err
		}
	}

	for _, acc := range accList {
		if err := accountService.IndexAccountFiles(acc); err != nil {
			log.Println("Fail to sync account", acc.Id.Hex(), "by error", err.Error())
		}
	}
	// TODO: should aggregate error here!
	return nil
}

func (s *ProjectService) ListAccounts(projectId string) ([]*iam.ServiceAccount, error) {
	var p entity.Project
	hex, _ := primitive.ObjectIDFromHex(projectId)
	if err := dao.Project().FindOne(context.Background(), bson.D{{"_id", hex}}).Decode(&p);
		err != nil {
		log.Println("No project found for id", projectId)
		return nil, err
	}

	var key []byte
	var admin entity.DriveAccount
	hex_, _ := primitive.ObjectIDFromHex(projectId)
	if err := dao.DriveAccount().FindOne(context.Background(), bson.D{
		{"projectId", hex_},
		{"type", "service_account_admin"},
	}).Decode(&admin); err != nil {
		log.Println("No admin account for project", projectId)
		key = []byte(p.AdminKey)
	} else {
		key = []byte(admin.Key)
	}

	is, err := helper.NewIamService(key)
	if err != nil {
		return nil, err
	}

	return is.ListServiceAccounts(p.ProjectId)
}

func (s *ProjectService) DeleteProject(projectId string) error {
	log.Println("Deleting project", projectId)
	pid, _ := primitive.ObjectIDFromHex(projectId)
	if ci, err := dao.FileIndex().DeleteMany(context.Background(), bson.D{
		{"projectId", pid},
	}); err != nil {
		log.Println("Fail to delete file indexes by error", err.Error())
		return err
	} else {
		log.Println("Deleted {} file_index records", ci.DeletedCount)
	}
	if ci, err := dao.DriveAccount().DeleteMany(context.Background(), bson.D{
		{"projectId", pid},
	}); err != nil {
		log.Println("Fail to delete drive accounts by error", err.Error())
		return err
	} else {
		log.Println("Deleted {} drive_account records", ci.DeletedCount)
	}
	if ci, err := dao.Project().DeleteMany(context.Background(), bson.D{
		{"_id", pid},
	}); err != nil {
		log.Println("Fail to delete drive accounts by error", err.Error())
		return err
	} else {
		log.Println("Deleted {} drive_account records", ci.DeletedCount)
	}
	log.Println("Successfully deleted project", pid.Hex())
	return nil
}

func (s *ProjectService) SyncProjectWithGoogle(projectId string) error {
	proj, err := s.GetProject(projectId)
	if err != nil {
		log.Println("Project not found by error", err.Error())
		return err
	}

	if ci, err := dao.FileIndex().DeleteMany(context.Background(), bson.D{
		{"projectId", proj.Id},
	}); err != nil {
		log.Println("Fail to delete file indexes by error", err.Error())
		return err
	} else {
		log.Println("Deleted {} file_index records", ci.DeletedCount)
	}
	if ci, err := dao.DriveAccount().DeleteMany(context.Background(), bson.D{
		{"projectId", proj.Id},
		{"type", "service_account"},
	}); err != nil {
		log.Println("Fail to delete drive accounts by error", err.Error())
		return err
	} else {
		log.Println("Deleted {} drive_account records", ci.DeletedCount)
	}

	is, err := s.GetIamService(proj)
	if err != nil {
		log.Println("Fail to get iam service for proj", projectId, "by error", err.Error())
		return err
	}
	remoteAccounts, err := is.ListServiceAccounts(proj.ProjectId)
	if err != nil {
		log.Println("Fail to get service accounts for proj", projectId)
		return err
	}
	for idx, acc := range remoteAccounts {
		if acc.UniqueId == is.KeyDetails.ClientId {
			log.Println("Ignore admin account", acc.UniqueId)
			continue
		}

		if err := is.RemoveExistingKeys(acc); err != nil {
			log.Println("No account key exists for service account", acc.Name)
		}
		key, err := is.CreateServiceAccountKey(acc)
		if err != nil {
			log.Println("Fail to create service account key by error", err.Error())
			return err
		}
		if en, err := s.InsertDriveAccount(proj, acc, key); err != nil {
			log.Println("Fail to insert drive account by error", err.Error())
			return err
		} else {
			if err := accountService.IndexAccountFiles(*en); err != nil {
				log.Println("Fail to index account's files by error", err.Error())
			}
		}
		log.Println("Synchronized", idx+1, "of", len(remoteAccounts), "accounts")
	}

	return nil
}

func (s *ProjectService) InsertDriveAccount(proj *entity.Project, sa *iam.ServiceAccount, key []byte) (*entity.DriveAccount, error) {
	newAcc := entity.DriveAccount{}
	if err := accountService.InitializeKey(&newAcc, key); err != nil {
		log.Println("Fail to parse service account key file by error", err.Error())
		return nil, err
	}
	newAcc.Name = sa.DisplayName
	newAcc.Owner = proj.Owner
	newAcc.ProjectId = proj.Id
	newAcc.Key = string(key)

	srv, err := helper.GetDriveService(key)
	if err != nil {
		log.Println("Fail to get drive service from account key by error", err.Error())
		return nil, err
	}
	tries := 0
	for {
		tries++
		quota, err := srv.GetQuotaUsage()
		if err != nil {
			time.Sleep(2 * time.Second)
		} else {
			newAcc.Usage = quota.Usage
			newAcc.Limit = quota.Limit
			newAcc.QuotaUpdateTimestamp = time.Now()
			break
		}
		if tries >= 30 {
			log.Println("Account may not ready at this moment. Tries:", tries)
			return nil, err
		}
	}

	if err := accountService.Save(&newAcc); err != nil {
		log.Println("Fail to save new service account to DB by error", err.Error())
		return nil, err
	}
	return &newAcc, nil
}

func (s *ProjectService) GetProject(id string) (*entity.Project, error) {
	var p entity.Project
	hex, _ := primitive.ObjectIDFromHex(id)
	if err := dao.Project().FindOne(context.Background(), bson.D{{"_id", hex}}).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *ProjectService) GetIamService(project *entity.Project) (*helper.IamService, error) {
	if project.AdminKey != "" {
		return helper.NewIamService([]byte(project.AdminKey))
	}
	account, err := accountService.FindAdminAccount(project.Id.Hex())
	if err != nil {
		log.Println("No admin account found for project", project.Id.Hex())
		return nil, err
	}
	return helper.NewIamService([]byte(account.Key))
}

func (s *ProjectService) SyncProjectQuota(projectId string) error {
	var accounts []entity.DriveAccount
	projectIdHex, _ := primitive.ObjectIDFromHex(projectId)
	if cursor, err := dao.DriveAccount().Find(context.Background(), bson.D{
		{"projectId", projectIdHex},
	}); err != nil {
		return err
	} else {
		if err := cursor.All(context.Background(), &accounts); err != nil {
			return err
		}
	}
	as := GetAccountService()
	for _, account := range accounts {
		if err := as.UpdateCachedQuotaByAccountId(account.Id.Hex()); err != nil {
			log.Println("Fail to update quota for account", account.Id.Hex(), "by error", err.Error())
		}
	}
	return nil
}

func (s *ProjectService) DisableProject(projectId string) error {
	return s.setProjectDisabled(projectId, true)
}

func (s *ProjectService) EnableProject(projectId string) error {
	return s.setProjectDisabled(projectId, false)
}

func (s *ProjectService) setProjectDisabled(projectId string, disabled bool) error {
	if !primitive.IsValidObjectID(projectId) {
		return errors.New("InvalidProjectId")
	}
	pid, _ := primitive.ObjectIDFromHex(projectId)
	if _, err := dao.ExecTransaction(func(sessCtx mongo.SessionContext) (interface{}, error) {
		log.Println("Disable the project", pid.Hex())
		if _, err := dao.Project().UpdateOne(context.Background(), bson.D{{"_id", pid}}, bson.D{
			{"$set", bson.D{
				{"disabled", disabled},
			}},
		}); err != nil {
			return err, nil
		}
		if _, err := dao.DriveAccount().UpdateMany(context.Background(), bson.D{{"projectId", pid}}, bson.D{
			{"$set", bson.D{
				{"disabled", disabled},
			}},
		}); err != nil {
			return nil, err
		}
		return nil, nil
	}); err != nil {
		return err
	}
	return nil
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
	account, err := createServiceAccount(iamSrv, project.ProjectId, accountName, accountName)
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
		log.Println("Retrieving quota usage for account", newAcc.ClientId)
		quota, err := srv.GetQuotaUsage()
		if err != nil {
			log.Println("Account may not ready at this moment. Error=", err.Error(), ". Tries:", tries)
			time.Sleep(2 * time.Second)
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
