package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/helper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/api/drive/v3"
	"log"
	"strconv"
	"sync"
	"time"
)

type AccountService struct {
	//accountCache map[string]*helper.DriveService
}

type KeyDetails struct {
	Type        string `json:"type"`
	ProjectId   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	ClientId    string `json:"client_id"`
}

func (s *AccountService) Save(account *entity.DriveAccount) error {
	account.Id = primitive.NewObjectID()
	return dao.DriveAccount().Insert(account)
}

func (s *AccountService) FindAll() ([]*entity.DriveAccount, error) {
	var list []*entity.DriveAccount
	err := dao.DriveAccount().FindAll(&list)
	return list, err
}

func (s *AccountService) FindAllByOwner(owner primitive.ObjectID) ([]*entity.DriveAccount, error) {
	var list []*entity.DriveAccount
	err := dao.DriveAccount().Find(bson.D{{"owner", owner}}, &list)
	return list, err
}

func (s *AccountService) FindAccounts(page int, size int, includeKey bool, owner string) ([]*entity.DriveAccount, bool, error) {
	var list []*entity.DriveAccount
	findOptions := options.Find()
	findOptions.SetLimit(int64(size + 1))
	findOptions.SetSkip(int64((page - 1) * size))
	findOptions.SetSort(bson.D{{"name", 1}})
	findOptions.SetProjection(bson.D{{"key", includeKey}})
	err := dao.DriveAccount().Template(func(col *mongo.Collection) error {
		ownerHex, _ := primitive.ObjectIDFromHex(owner)
		find, err := col.Find(context.TODO(), bson.D{{"owner", ownerHex}}, findOptions)
		if err != nil {
			return err
		}
		return find.All(context.TODO(), &list)
	})
	hasMore := false
	if len(list) == size+1 {
		hasMore = true
		list = list[:len(list)-1]
	}
	return list, hasMore, err
}

type AccountLookup struct {
	Id          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Desc        string             `json:"desc" bson:"desc"`
	Type        string             `json:"type" bson:"type"`
	Key         string             `json:"-" bson:"key"`
	ClientEmail string             `json:"clientEmail" bson:"clientEmail"`
	ClientId    string             `json:"clientId" bson:"clientId"`
	Usage       int64              `json:"usage" bson:"usage"`
	Limit       int64              `json:"limit" bson:"limit"`
	Project     entity.Project     `json:"project" bson:"project"`
	Files       []*helper.File     `json:"files"`
}

func (s *AccountService) FindAccountLookup(id string, userId string) (*AccountLookup, error) {
	var acc AccountLookup
	hexId, _ := primitive.ObjectIDFromHex(id)
	hexUserId, _ := primitive.ObjectIDFromHex(userId)
	cursor, err := dao.RawCollection("drive_account").Aggregate(context.Background(), bson.D{
		{"$match", bson.D{
			{"_id", hexId},
			{"owner", hexUserId},
		}},
		{"$lookup", bson.D{
			{"from", "project"},
			{"localField", "projectId"},
			{"foreignField", "_id"},
			{"as", "projects"},
		}},
		{"$addFields", bson.D{
			{"project", bson.D{
				{"$arrayElemAt", []interface{}{"$projects", 0}},
			}},
		}},
		{
			"$project", bson.D{
			{"projects", 0},
		},
		},
	})
	if err != nil {
		return nil, err
	}
	if err := cursor.Decode(&acc); err != nil {
		return nil, err
	}

	srv, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		return nil, err
	}
	files, err := srv.ListFiles(1, 50)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		file.AccountId = id
	}
	acc.Files = files

	return &acc, err
}

func (s *AccountService) FindAccount(id string) (*entity.DriveAccount, error) {
	var acc entity.DriveAccount
	hex, _ := primitive.ObjectIDFromHex(id)
	err := dao.DriveAccount().FindId(hex, &acc)
	return &acc, err
}

func (s *AccountService) FindAccountById(id primitive.ObjectID, owner primitive.ObjectID) (*entity.DriveAccount, error) {
	var acc entity.DriveAccount
	err := dao.DriveAccount().FindOne(bson.D{
		{"_id", id},
		{"owner", owner},
	}, &acc)
	return &acc, err
}

func (s *AccountService) InitializeKey(acc *entity.DriveAccount, key []byte) error {
	var kd KeyDetails
	err := json.Unmarshal(key, &kd)
	if err != nil {
		return err
	}
	acc.Key = string(key)
	acc.ClientId = kd.ClientId
	acc.ClientEmail = kd.ClientEmail
	acc.Type = kd.Type

	return nil
}

func (s *AccountService) UpdateKey(id string, key []byte) error {
	//var acc entity.DriveAccount
	//err := dao.DriveAccount().FindId(primitive.ObjectIDFromHex(id), &acc)
	//if err != nil {
	//	return err
	//}
	//err = s.InitializeKey(&acc, key)
	//if err != nil {
	//	return err
	//}
	//return dao.DriveAccount().UpdateId(primitive.ObjectIDFromHex(id), &acc)
	// TODO
	return nil
}

func (s *AccountService) UpdateCachedQuotaByAccountId(accountId string) error {
	//var acc entity.DriveAccount
	//if err := dao.DriveAccount().FindId(primitive.ObjectIDFromHex(accountId), &acc); err != nil {
	//	return err
	//}
	//return s.UpdateCachedQuota(&acc)
	// TODO
	return nil
}

func (s *AccountService) UpdateCachedQuotaByAccountIdAndAdditionalSize(accountId string, addedSize int64) error {
	//var acc entity.DriveAccount
	//if err := dao.DriveAccount().FindId(primitive.ObjectIDFromHex(accountId), &acc); err != nil {
	//	return err
	//}
	////return s.UpdateCachedQuota(&acc)
	//updatedAt := time.Now()
	//return dao.DriveAccount().Update(
	//	bson.M{"_id": acc.Id},
	//	bson.M{
	//		"$set": bson.M{
	//			"usage":                acc.Usage + addedSize,
	//			"available":            acc.Limit - addedSize,
	//			"quotaUpdateTimestamp": updatedAt,
	//		},
	//	})
	// TODO
	return nil
}

func (s *AccountService) UpdateCachedQuota(acc *entity.DriveAccount) error {
	//driveService, err := helper.GetDriveService([]byte(acc.Key))
	//if err != nil {
	//	return err
	//}
	//quota, err := driveService.GetQuotaUsage()
	//if err != nil {
	//	return err
	//}
	//updatedAt := time.Now()
	//acc.Usage = quota.Usage
	//acc.Limit = quota.Limit
	//acc.Available = quota.Limit - quota.Usage
	//acc.QuotaUpdateTimestamp = updatedAt
	//return dao.DriveAccount().Update(
	//	bson.M{"_id": acc.Id},
	//	bson.M{
	//		"$set": bson.M{
	//			"usage":                quota.Usage,
	//			"limit":                quota.Limit,
	//			"available":            quota.Limit - quota.Usage,
	//			"quotaUpdateTimestamp": updatedAt,
	//		},
	//	})
	// TODO
	return nil
}

type FileLookup struct {
	Id      primitive.ObjectID    `json:"_id" bson:"_id"`
	DriveId string                `json:"driveId" bson:"driveId"`
	Name    string                `json:"name" bson:"name"`
	Account []entity.DriveAccount `json:"account" bson:"account"`
}

type FileAggregateResult struct {
	Id         primitive.ObjectID `json:"_id" bson:"_id"`
	DriveId    string             `json:"driveId" bson:"driveId"`
	Name       string             `json:"name" bson:"name"`
	AccountKey string             `json:"accountKey" bson:"accountKey"`
}

func (s *AccountService) GetDownloadLinkByFileId(fileId string) (*drive.File, *helper.DownloadDetails, error) {
	res := FileAggregateResult{}
	fileIdHex, _ := primitive.ObjectIDFromHex(fileId)
	cursor, err := dao.RawCollection("file").Aggregate(context.Background(), bson.D{
		{"$match", bson.D{{"_id", fileIdHex}}},
		{"$lookup", bson.D{
			{"from", "drive_account"},
			{"localField", "driveAccount"},
			{"foreignField", "_id"},
			{"as", "accounts"},
		}},
		{"$unwind", bson.D{{"path", "$accounts"}}},
		{"$project", bson.D{
			{"driveId", 1},
			{"name", 1},
			{"accountKey", "$account.key"},
		}},
	})
	if err != nil {
		return nil, nil, err
	}
	if err := cursor.Decode(&res); err != nil {
		return nil, nil, err
	}
	srv, err := helper.GetDriveService([]byte(res.AccountKey))
	if err != nil {
		return nil, nil, err
	}
	gFile, link, err := srv.GetDownloadLink(res.DriveId)
	if err != nil {
		fmt.Println("fail to get download link", err.Error())
		return nil, nil, err
	}
	return gFile, link, nil
}

//
//func getDriveService(driveId primitive.ObjectID) (*helper.DriveService, error) {
//	acc := entity.DriveAccount{}
//	if err := ; err != nil {
//		return nil, err
//	}
//	return helper.GetDriveService([]byte(acc.Key))
//
//}

//func (s *AccountService) GetDownloadLink(file *entity.DriveFile) (*drive.File, string, error) {
//
//	driveService := s.accountCache[file.DriveAccount.Hex()]
//	gFile, link, err := driveService.GetDownloadLink(file.DriveFileId)
//	if err != nil {
//		fmt.Println("fail to get download link", err.Error())
//		return nil, "", err
//	}
//	return gFile, link, nil
//}

func (s *AccountService) UpdateAllAccountQuota() error {
	fmt.Println("Updating account quota...")
	all, err := s.FindAll()
	if err != nil {
		fmt.Println("fail to query all account")
	}
	wg := sync.WaitGroup{}
	for _, acc := range all {
		wg.Add(1)
		go func(id string, name string) {
			err := s.UpdateCachedQuota(acc)
			if err != nil {
				fmt.Println("fail to update quota for", id, name, "error", err.Error())
			}
			wg.Done()
		}(acc.Id.Hex(), acc.Name)
	}
	wg.Wait()
	fmt.Println("finished update account quota")
	return nil
}

func (s *AccountService) UpdateAccountQuotaByOwner(owner primitive.ObjectID) error {
	log.Println("Updating account quota for user", owner.Hex())
	all, err := s.FindAllByOwner(owner)
	if err != nil {
		log.Println("fail to query all account")
		return err
	}
	for _, acc := range all {
		err := s.UpdateCachedQuota(acc)
		if err != nil {
			fmt.Println("fail to update quota for", acc.Id.Hex(), acc.Name, "error", err.Error())
		}
	}
	fmt.Println("finished update account quota")
	return nil
}

var accountService *AccountService

func GetAccountService() *AccountService {
	if accountService == nil {
		accountService = &AccountService{
		}
	}
	return accountService
}

func (s *AccountService) GetAccountCount() int64 {
	n, _ := dao.RawCollection("drive_account").CountDocuments(context.Background(), nil)
	return n
}
func (s *AccountService) GetAccessToken(acc *entity.DriveAccount) (string, error) {
	srv, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		return "", err
	}
	return srv.GetAccessToken()
}

func (s *AccountService) CreateServiceAccount(projectId string, userId string) (*entity.DriveAccount, error) {
	admin, err := s.FindAdminAccount(projectId)
	if err != nil {
		log.Println("Unable to find admin account for this project by error", err.Error())
		return nil, err
	}

	key, err := parseKeyDetails([]byte(admin.Key))
	if err != nil {
		log.Println("Fail to parse key of admin account")
		return nil, err
	}

	iamService, err := helper.NewIamService([]byte(admin.Key))
	if err != nil {
		log.Println("Fail to initialize IAM service with admin key by error", err.Error())
		return nil, err
	}

	serviceAccountId := "sa-" + strconv.FormatInt(time.Now().Unix(), 16)
	account, err := iamService.CreateServiceAccount(key.ProjectId, serviceAccountId, "Automate account")
	if err != nil {
		log.Println("Fail to create service account by error", err.Error())
		return nil, err
	}
	saKey, err := iamService.CreateServiceAccountKey(account)
	if err != nil {
		log.Println("Fail to generate service account key file by error", err.Error())
		return nil, err
	}
	userIdHex, _ := primitive.ObjectIDFromHex(userId)
	projectIdHex, _ := primitive.ObjectIDFromHex(projectId)
	acc := &entity.DriveAccount{
		Id:                   primitive.NewObjectID(),
		Name:                 serviceAccountId,
		Desc:                 account.DisplayName,
		Type:                 "service_account",
		ClientEmail:          account.Email,
		ClientId:             account.Oauth2ClientId,
		Key:                  string(saKey),
		Usage:                0,
		Limit:                0,
		Owner:                userIdHex,
		ProjectId:            projectIdHex,
		QuotaUpdateTimestamp: time.Time{},
	}

	if err := dao.DriveAccount().Insert(acc); err != nil {
		return nil, err
	}

	go s.UpdateCachedQuotaByAccountId(acc.Id.Hex())

	return acc, nil
}

func (s *AccountService) FindAdminAccount(projectId string) (*entity.DriveAccount, error) {
	var admin entity.DriveAccount
	projectIdHex, _ := primitive.ObjectIDFromHex(projectId)
	if err := dao.DriveAccount().FindOne(bson.D{
		{"projectId", projectIdHex},
		{"type", "service_account_admin"},
	}, &admin); err != nil {
		var project entity.Project
		if err := dao.Project().FindId(projectIdHex, &project); err != nil {
			return nil, err
		}

		return migrateAdminAccount(project)
	} else {
		return &admin, nil
	}
}

func migrateAdminAccount(project entity.Project) (*entity.DriveAccount, error) {
	log.Println("Migrate admin account for project", project.Id.Hex())
	kd := KeyDetails{}
	if err := json.Unmarshal([]byte(project.AdminKey), &kd); err != nil {
		return nil, err
	}
	accountId := primitive.NewObjectID()
	acc := entity.DriveAccount{
		Id:          accountId,
		ProjectId:   project.Id,
		Name:        "admin-account",
		Key:         project.AdminKey,
		Desc:        "Admin Account",
		Type:        "service_account_admin",
		ClientEmail: kd.ClientEmail,
		ClientId:    kd.ClientId,
		Owner:       project.Owner,
	}

	if err := dao.DriveAccount().Insert(&acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

type FileIndex struct {
	Id           primitive.ObjectID `json:"id" bson:"_id"`
	FileId       string             `json:"fileId" bson:"fileId"`
	Name         string             `json:"name" bson:"name"`
	Size         int64              `json:"size" bson:"size"`
	MimeType     string             `json:"mimeType" bson:"mimeType"`
	AccountId    primitive.ObjectID `json:"accountId" bson:"accountId"`
	Owner        primitive.ObjectID `json:"owner" bson:"owner"`
	ProjectId    primitive.ObjectID `json:"projectId" bson:"projectId"`
	CreatedTime  time.Time          `json:"createdTime" bson:"createdTime"`
	ModifiedTime time.Time          `json:"modifiedTime" bson:"modifiedTime"`
	SyncTime     time.Time          `json:"syncTime" bson:"syncTime"`
}

func (s *AccountService) IndexAccountFiles(acc entity.DriveAccount) error {
	ds, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		log.Println("Account", acc.Id.Hex(), "Fail to get drive service from key by error", err.Error())
		return err
	}

	if _, err := dao.RawCollection("drive_account").DeleteMany(context.Background(), bson.D{
		{"accountId", acc.Id},
	}); err != nil {
		log.Println("Fail to remove old files index")
	}

	page := 1
	size := 500
	for {
		files, err := ds.ListFiles(page, int64(size))
		if err != nil {
			log.Println("Account", acc.Id.Hex(), "Fail to list files in account by error", err.Error())
			return err
		}

		for _, file := range files {
			log.Println(file.Id, file.Name, file.AccountId, file.MimeType)
			ct, _ := time.Parse("2006-01-02T15:04:05Z", file.CreatedTime)
			mt, _ := time.Parse("2006-01-02T15:04:05Z", file.ModifiedTime)

			f := FileIndex{
				Id:           primitive.NewObjectID(),
				FileId:       file.Id,
				Name:         file.Name,
				Size:         file.Size,
				MimeType:     file.MimeType,
				AccountId:    acc.Id,
				Owner:        acc.Owner,
				ProjectId:    acc.ProjectId,
				CreatedTime:  ct,
				ModifiedTime: mt,
				SyncTime:     time.Now(),
			}
			if err := dao.FileIndex().Insert(f); err != nil {
				log.Println("Fail to insert file index")
				return err
			}
		}

		if len(files) < size {
			break
		}
		page = page + 1
	}

	return nil

}

type FileFavorite struct {
	Id     primitive.ObjectID `json:"id" bson:"_id"`
	FileId string             `json:"fileId" bson:"fileId"`
	UserId primitive.ObjectID `json:"userId" bson:"userId"`
}

func (s *AccountService) SetFileFavorite(userId, accountId, fileId string, favorite bool) (*FileFavorite, error) {
	var existing FileFavorite
	userIdHex, _ := primitive.ObjectIDFromHex(userId)
	if err := dao.FileFavorite().FindOne(bson.D{
		{"fileId", fileId},
		{"userId", userIdHex},
	}, &existing); err == nil {
		return &existing, nil
	}

	fv := FileFavorite{
		Id:     primitive.NewObjectID(),
		UserId: userIdHex,
		FileId: fileId,
	}
	if err := dao.FileFavorite().Insert(fv); err != nil {
		return nil, err
	}
	return &fv, nil
}

func (s *AccountService) SyncFile(userId string, accountId string, cloudFile drive.File) (*FileIndex, error) {
	var acc entity.DriveAccount
	accountIdHex, _ := primitive.ObjectIDFromHex(accountId)
	if err := dao.DriveAccount().FindId(accountIdHex, &acc); err != nil {
		log.Println("SyncFile", "failed by error", err.Error())
		return nil, err
	}
	ct, _ := time.Parse("2006-01-02T15:04:05Z", cloudFile.CreatedTime)
	mt, _ := time.Parse("2006-01-02T15:04:05Z", cloudFile.ModifiedTime)
	f := FileIndex{
		Id:           primitive.NewObjectID(),
		FileId:       cloudFile.Id,
		Name:         cloudFile.Name,
		Size:         cloudFile.Size,
		MimeType:     cloudFile.MimeType,
		AccountId:    acc.Id,
		Owner:        acc.Owner,
		ProjectId:    acc.ProjectId,
		CreatedTime:  ct,
		ModifiedTime: mt,
		SyncTime:     time.Now(),
	}
	if err := dao.FileIndex().Insert(f); err != nil {
		log.Println("SyncFile", "Fail to insert file index by error", err.Error())
		return nil, err
	}
	return &f, nil
}

func (s *AccountService) SyncFileById(userId string, accountId string, fileId string) (*FileIndex, error) {
	var existing []FileIndex
	if err := dao.FileIndex().Find(bson.D{{"fileId", fileId}}, &existing); err != nil {
		log.Println("SyncFileById", "Fail to query if file already sync", err.Error())
		return nil, err
	}
	if len(existing) > 0 {
		log.Println("File already synchronized")
		return &existing[0], nil
	}
	var acc entity.DriveAccount
	accountIdHex, _ := primitive.ObjectIDFromHex(accountId)
	if err := dao.DriveAccount().FindId(accountIdHex, &acc); err != nil {
		log.Println("SyncFileById", "failed by error", err.Error())
		return nil, err
	}
	ds, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		log.Println("SyncFileById", "Account", acc.Id.Hex(), "Fail to get drive service from key by error", err.Error())
		return nil, err
	}
	cloudFile, err := ds.GetFile(fileId)
	if err != nil {
		log.Println("SyncFileById", "Account", acc.Id.Hex(), "Fail to get cloud file by error", err.Error())
		return nil, err
	}
	ct, _ := time.Parse("2006-01-02T15:04:05Z", cloudFile.CreatedTime)
	mt, _ := time.Parse("2006-01-02T15:04:05Z", cloudFile.ModifiedTime)
	f := FileIndex{
		Id:           primitive.NewObjectID(),
		FileId:       cloudFile.Id,
		Name:         cloudFile.Name,
		Size:         cloudFile.Size,
		MimeType:     cloudFile.MimeType,
		AccountId:    acc.Id,
		Owner:        acc.Owner,
		ProjectId:    acc.ProjectId,
		CreatedTime:  ct,
		ModifiedTime: mt,
		SyncTime:     time.Now(),
	}
	if err := dao.FileIndex().Insert(f); err != nil {
		log.Println("SyncFileById", "Fail to insert file index by error", err.Error())
		return nil, err
	}
	return &f, nil
}

func (s *AccountService) ListFile(accountId string) ([]*helper.File, error) {
	acc, err := s.FindAccount(accountId)
	if err != nil {
		return nil, err
	}
	ds, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		log.Println("SyncFileById", "Account", acc.Id.Hex(), "Fail to get drive service from key by error", err.Error())
		return nil, err
	}

	//accountService.FindAdminAccount()
	return ds.ListFiles(1, 1000)
}
