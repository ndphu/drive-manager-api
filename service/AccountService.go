package service

import (
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	driveApi "github.com/ndphu/google-api-helper"
	"sync"
	"time"
)

type AccountService struct {
	accountCache map[string]*driveApi.DriveService
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

func (s *AccountService) FindAccounts(page int, size int) ([]*entity.DriveAccount, error) {
	var list []*entity.DriveAccount
	err := dao.Collection("drive_account").
		Find(bson.M{}).
		//Select(bson.M{"key": 0}).
		Skip((page - 1) * size).
		Limit(size).
		All(&list)
	return list, err
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
	acc.ProjectId = kd.ProjectId
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
func (s *AccountService) UpdateCachedQuota(id string) error {
	driveService := s.accountCache[id]
	quota, err := driveService.GetQuotaUsage()
	if err != nil {
		return err
	}
	return dao.Collection("drive_account").Update(
		bson.M{"_id": bson.ObjectIdHex(id)},
		bson.M{
			"$set": bson.M{
				"usage":                quota.Usage,
				"limit":                quota.Limit,
				"quotaUpdateTimestamp": time.Now(),
			},
		})
}

type FileLookup struct {
	Id      bson.ObjectId         `json:"_id" bson:"_id"`
	DriveId string                `json:"driveId" bson:"driveId"`
	Name    string                `json:"name" bson:"name"`
	Account []entity.DriveAccount `json:"account" bson:"account"`
}

func (s *AccountService) GetDownloadLinkByFileId(fileId string) (string, error) {
	file := entity.DriveFile{}
	err := dao.Collection("file").FindId(bson.ObjectIdHex(fileId)).One(&file)
	if err != nil {
		return "", err
	}
	driveService := s.accountCache[file.DriveAccount.Hex()]
	link, err := driveService.GetDownloadLink(file.DriveFileId)
	if err != nil {
		fmt.Println("fail to get download link", err.Error())
		return "", err
	}
	return link, nil
}
func (s *AccountService) GetDownloadLink(file *entity.DriveFile) (string, error) {
	driveService := s.accountCache[file.DriveAccount.Hex()]
	link, err := driveService.GetDownloadLink(file.DriveFileId)
	if err != nil {
		fmt.Println("fail to get download link", err.Error())
		return "", err
	}
	return link, nil
}
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
			accountCache: make(map[string]*driveApi.DriveService, 0),
		}
		accountService.UpdateAccountCache()
	}
	return accountService, nil
}

func (s *AccountService) UpdateAccountCache() error {
	all, err := accountService.FindAll()
	if err != nil {
		return err
	}
	for _, acc := range all {
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
