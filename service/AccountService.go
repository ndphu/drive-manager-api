package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	driveApi "github.com/ndphu/google-api-helper"
	"google.golang.org/api/drive/v3"
	"log"
	"sync"
	"time"
)

type AccountService struct {
	//accountCache map[string]*driveApi.DriveService
}

type KeyDetails struct {
	Type        string `json:"type"`
	ProjectId   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	ClientId    string `json:"client_id"`
}

func (s *AccountService) Save(account *entity.DriveAccount) error {
	account.Id = bson.NewObjectId();
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

func (s *AccountService) FindAccount(id string) (*entity.DriveAccount, error) {
	var acc entity.DriveAccount
	err := dao.Collection("drive_account").FindId(bson.ObjectIdHex(id)).One(&acc)
	return &acc, err
}

func (s *AccountService) InitializeKey(acc *entity.DriveAccount, key []byte) (error) {
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

func (s *AccountService) UpdateKey(id string, key []byte) (error) {
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
	driveService, err := driveApi.GetDriveService([]byte(acc.Key))
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

func (s *AccountService) GetDownloadLinkByFileId(fileId string) (*drive.File, string, error) {
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
		return nil, "", err
	}
	srv, err := driveApi.GetDriveService([]byte(res.AccountKey))
	if err != nil {
		return nil, "", err
	}
	gFile, link, err := srv.GetDownloadLink(res.DriveId)
	if err != nil {
		fmt.Println("fail to get download link", err.Error())
		return nil, "", err
	}
	return gFile, link, nil
}

func getDriveService(driveId bson.ObjectId) (*driveApi.DriveService, error) {
	acc := entity.DriveAccount{}
	if err := dao.Collection("drive_account").FindId(driveId).One(&acc); err != nil {
		return nil, err
	}
	return driveApi.GetDriveService([]byte(acc.Key))

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
	wg := sync.WaitGroup{}
	for _, acc := range all {
		wg.Add(1)
		go func(id string, name string) {
			err := s.UpdateCachedQuota(id)
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

var accountService *AccountService

func GetAccountService() (*AccountService, error) {
	if accountService == nil {
		accountService = &AccountService{
		}
	}
	return accountService, nil
}


func (s *AccountService) UpdateAccountCacheByOwner(owner bson.ObjectId) error {
	var list []*entity.DriveAccount
	err := dao.Collection("drive_account").Find(bson.M{"owner": owner}).All(&list)
	if err != nil {
		return err
	}
	for _, acc := range list {
		if len(acc.Key) == 0 {
			continue
		}
		driveService, err := driveApi.GetDriveService([]byte(acc.Key))
		if err != nil {
			return err
		}
		accountService.accountCache[acc.Id.Hex()] = driveService
	}
	fmt.Println("cached", len(accountService.accountCache), "accounts")
	return nil
}

func (s *AccountService) GetAccountCount() int {
	n, _ := dao.Collection("drive_account").Count()
	return n
}
