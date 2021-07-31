package service

import (
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/helper"
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
	account.Id = bson.NewObjectId()
	return dao.Collection("drive_account").Insert(account)
}

func (s *AccountService) FindAll() ([]*entity.DriveAccount, error) {
	var list []*entity.DriveAccount
	err := dao.Collection("drive_account").Find(bson.M{}).All(&list)
	return list, err
}

func (s *AccountService) FindAllByOwner(owner bson.ObjectId) ([]*entity.DriveAccount, error) {
	var list []*entity.DriveAccount
	err := dao.Collection("drive_account").Find(bson.M{"owner": owner}).All(&list)
	return list, err
}

func (s *AccountService) FindAccounts(page int, size int, includeKey bool, owner string) ([]*entity.DriveAccount, bool, error) {
	var list []*entity.DriveAccount
	err := dao.Collection("drive_account").
		Find(bson.M{"owner": bson.ObjectIdHex(owner)}).
		Select(bson.M{"key": includeKey}).
		Sort("name").
		Skip((page - 1) * size).
		Limit(size + 1).
		All(&list)
	hasMore := false
	if len(list) == size+1 {
		hasMore = true
		list = list[:len(list)-1]
	}
	return list, hasMore, err
}

type AccountLookup struct {
	Id          bson.ObjectId  `json:"id" bson:"_id"`
	Name        string         `json:"name" bson:"name"`
	Desc        string         `json:"desc" bson:"desc"`
	Type        string         `json:"type" bson:"type"`
	Key         string         `json:"-" bson:"key"`
	ClientEmail string         `json:"clientEmail" bson:"clientEmail"`
	ClientId    string         `json:"clientId" bson:"clientId"`
	Usage       int64          `json:"usage" bson:"usage"`
	Limit       int64          `json:"limit" bson:"limit"`
	Project     entity.Project `json:"project" bson:"project"`
	Files       []*helper.File `json:"files"`
}

func (s *AccountService) FindAccountLookup(id string, userId string) (*AccountLookup, error) {
	var acc AccountLookup
	err := dao.Collection("drive_account").Pipe([]bson.M{
		{"$match": bson.M{
			"_id":   bson.ObjectIdHex(id),
			"owner": bson.ObjectIdHex(userId),
		}},
		{"$lookup": bson.M{
			"from":         "project",
			"localField":   "projectId",
			"foreignField": "_id",
			"as":           "projects",
		}},
		{"$addFields": bson.M{
			"project": bson.M{
				"$arrayElemAt": []interface{}{"$projects", 0},
			},
		}},
		{
			"$project": bson.M{
				"projects": 0,
			},
		},
	}).One(&acc)
	if err != nil {
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
	err := dao.Collection("drive_account").FindId(bson.ObjectIdHex(id)).One(&acc)
	return &acc, err
}

func (s *AccountService) FindAccountById(id bson.ObjectId, owner bson.ObjectId) (*entity.DriveAccount, error) {
	var acc entity.DriveAccount
	err := dao.Collection("drive_account").Find(bson.M{
		"_id":   id,
		"owner": owner,
	}).One(&acc)
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
	var acc entity.DriveAccount
	err := dao.Collection("drive_account").FindId(bson.ObjectIdHex(id)).One(&acc)
	if err != nil {
		return err
	}
	err = s.InitializeKey(&acc, key)
	if err != nil {
		return err
	}
	return dao.Collection("drive_account").UpdateId(bson.ObjectIdHex(id), &acc)
}
func (s *AccountService) UpdateCachedQuota(acc *entity.DriveAccount) error {
	driveService, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		return err
	}
	quota, err := driveService.GetQuotaUsage()
	if err != nil {
		return err
	}
	updatedAt := time.Now()
	acc.Usage = quota.Usage
	acc.Limit = quota.Limit
	acc.QuotaUpdateTimestamp = updatedAt
	return dao.Collection("drive_account").Update(
		bson.M{"_id": acc.Id},
		bson.M{
			"$set": bson.M{
				"usage":                quota.Usage,
				"limit":                quota.Limit,
				"quotaUpdateTimestamp": updatedAt,
			},
		})
}

type FileLookup struct {
	Id      bson.ObjectId         `json:"_id" bson:"_id"`
	DriveId string                `json:"driveId" bson:"driveId"`
	Name    string                `json:"name" bson:"name"`
	Account []entity.DriveAccount `json:"account" bson:"account"`
}

type FileAggregateResult struct {
	Id         bson.ObjectId `json:"_id" bson:"_id"`
	DriveId    string        `json:"driveId" bson:"driveId"`
	Name       string        `json:"name" bson:"name"`
	AccountKey string        `json:"accountKey" bson:"accountKey"`
}

func (s *AccountService) GetDownloadLinkByFileId(fileId string) (*drive.File, *helper.DownloadDetails, error) {
	res := FileAggregateResult{}
	if err := dao.Collection("file").Pipe([]bson.M{
		{"$match": bson.M{"_id": bson.ObjectIdHex(fileId)}},
		{"$lookup": bson.M{
			"from":         "drive_account",
			"localField":   "driveAccount",
			"foreignField": "_id",
			"as":           "accounts",
		}},
		{"$unwind": bson.M{"path": "$accounts"}},
		{"$project": bson.M{
			"driveId":    1,
			"name":       1,
			"accountKey": "$account.key",
		}},
	}).One(&res); err != nil {
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

func getDriveService(driveId bson.ObjectId) (*helper.DriveService, error) {
	acc := entity.DriveAccount{}
	if err := dao.Collection("drive_account").FindId(driveId).One(&acc); err != nil {
		return nil, err
	}
	return helper.GetDriveService([]byte(acc.Key))

}

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

func (s *AccountService) UpdateAccountQuotaByOwner(owner bson.ObjectId) error {
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

func (s *AccountService) GetAccountCount() int {
	n, _ := dao.Collection("drive_account").Count()
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
	admin, err := s.FindAdminAccount(projectId, userId)
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

	acc := &entity.DriveAccount{
		Id:                   bson.NewObjectId(),
		Name:                 serviceAccountId,
		Desc:                 account.DisplayName,
		Type:                 "service_account",
		ClientEmail:          account.Email,
		ClientId:             account.Oauth2ClientId,
		Key:                  string(saKey),
		Usage:                0,
		Limit:                0,
		Owner:                bson.ObjectIdHex(userId),
		ProjectId:            bson.ObjectIdHex(projectId),
		QuotaUpdateTimestamp: time.Time{},
	}

	if err := dao.Collection("drive_account").Insert(acc); err != nil {
		return nil, err
	}

	return acc, nil
}

func (s *AccountService) FindAdminAccount(projectId string, userId string) (*entity.DriveAccount, error) {
	var admin entity.DriveAccount
	if err := dao.Collection("drive_account").Find(bson.M{
		"projectId": bson.ObjectIdHex(projectId),
		"owner":     bson.ObjectIdHex(userId),
		"type":      "service_account_admin",
	}).One(&admin); err != nil {
		return nil, err
	}
	return &admin, nil
}

type FileIndex struct {
	Id           bson.ObjectId `json:"id" bson:"_id"`
	FileId       string        `json:"fileId" bson:"fileId"`
	Name         string        `json:"name" bson:"name"`
	Size         int64         `json:"size" bson:"size"`
	MimeType     string        `json:"mimeType" bson:"mimeType"`
	AccountId    bson.ObjectId `json:"accountId" bson:"accountId"`
	Owner        bson.ObjectId `json:"owner" bson:"owner"`
	ProjectId    bson.ObjectId `json:"projectId" bson:"projectId"`
	CreatedTime  time.Time     `json:"createdTime" bson:"createdTime"`
	ModifiedTime time.Time     `json:"modifiedTime" bson:"modifiedTime"`
	SyncTime     time.Time     `json:"syncTime" bson:"syncTime"`
}

func (s *AccountService) IndexAccountFiles(acc entity.DriveAccount) error {
	ds, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		log.Println("Account", acc.Id.Hex(), "Fail to get drive service from key by error", err.Error())
		return err
	}

	if info, err := dao.Collection("file_index").RemoveAll(bson.M{
		"accountId": acc.Id,
	}); err != nil {
		log.Println("Fail to remove old files index")
	} else {
		log.Printf("Change info %v\n", info)
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
				Id:           bson.NewObjectId(),
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
			if err := dao.Collection("file_index").Insert(f); err != nil {
				log.Println("Fail to insert file index")
				return err
			}
		}

		if len(files) < size {
			log.Println("Account", acc.Id.Hex(), "Looped to all file")
			break
		}
		page = page + 1
	}

	return nil

}

type FileFavorite struct {
	Id     bson.ObjectId `json:"id" bson:"_id"`
	FileId string        `json:"fileId" bson:"fileId"`
	UserId bson.ObjectId `json:"userId" bson:"userId"`
}

func (s *AccountService) SetFileFavorite(userId, accountId, fileId string, favorite bool) (*FileFavorite, error) {
	var existing FileFavorite
	if err := dao.Collection("file_favorite").Find(bson.M{
		"fileId": fileId,
		"userId": bson.ObjectIdHex(userId),
	}).One(&existing); err == nil {
		return &existing, nil
	}

	fv := FileFavorite{
		Id:     bson.NewObjectId(),
		UserId: bson.ObjectIdHex(userId),
		FileId: fileId,
	}
	if err := dao.Collection("file_favorite").Insert(fv); err != nil {
		return nil, err
	}
	return &fv, nil
}

func (s *AccountService) SyncFile(userId string, accountId string, fileId string) (*FileIndex, error) {
	var existing []FileIndex
	if err := dao.Collection("file_index").Find(bson.M{"fileId": fileId}).Limit(1).All(&existing); err != nil {
		log.Println("SyncFile", "Fail to query if file already sync", err.Error())
		return nil, err
	}
	if len(existing) > 0 {
		log.Println("File already synchronized")
		return &existing[0], nil
	}
	var acc entity.DriveAccount
	if err := dao.Collection("drive_account").FindId(bson.ObjectIdHex(accountId)).One(&acc); err != nil {
		log.Println("SyncFile", "failed by error", err.Error())
		return nil, err
	}
	ds, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		log.Println("SyncFile", "Account", acc.Id.Hex(), "Fail to get drive service from key by error", err.Error())
		return nil, err
	}
	cloudFile, err := ds.GetFile(fileId)
	if err != nil {
		log.Println("SyncFile", "Account", acc.Id.Hex(), "Fail to get cloud file by error", err.Error())
		return nil, err
	}
	ct, _ := time.Parse("2006-01-02T15:04:05Z", cloudFile.CreatedTime)
	mt, _ := time.Parse("2006-01-02T15:04:05Z", cloudFile.ModifiedTime)
	f := FileIndex{
		Id:           bson.NewObjectId(),
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
	if err := dao.Collection("file_index").Insert(f); err != nil {
		log.Println("SyncFile", "Fail to insert file index by error", err.Error())
		return nil, err
	}
	return &f, nil
}
