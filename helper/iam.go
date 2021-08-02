package helper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"log"
)

type IamService struct {
	Service    *iam.Service
	Config     *jwt.Config
	KeyDetails KeyDetails
}

func NewIamService(key []byte) (*IamService, error) {
	var kd KeyDetails
	if err := json.Unmarshal(key, &kd); err != nil {
		log.Println("Fail to parse admin key", err.Error())
		return nil, err
	}
	config, err := google.JWTConfigFromJSON(key, iam.CloudPlatformScope)
	if err != nil {
		log.Println("Fail to parse admin key", err.Error())
		return nil, err
	}

	service, err := iam.NewService(context.Background(), option.WithTokenSource(config.TokenSource(context.Background())))
	if err != nil {
		log.Println("Fail to initialize Google IAM service instance by error", err.Error())
		return nil, err
	}
	return &IamService{
		Service: service,
		Config: config,
		KeyDetails: kd,
	}, nil
}

func (s *IamService) CreateServiceAccount(googleProjectId, accountId, displayName string) (*iam.ServiceAccount, error) {
	req := iam.CreateServiceAccountRequest{}
	req.AccountId = accountId
	req.ServiceAccount = &iam.ServiceAccount{
		DisplayName: displayName,
	}
	return s.Service.Projects.ServiceAccounts.Create("projects/"+googleProjectId, &req).Do()
}

func (s *IamService) CreateServiceAccountKey(account *iam.ServiceAccount) ([]byte, error) {
	key, err := s.Service.Projects.ServiceAccounts.Keys.Create("projects/-/serviceAccounts/"+account.Email, &iam.CreateServiceAccountKeyRequest{}).Do()
	if err != nil {
		return nil, err
	}
	keyBytes, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return nil, err
	}
	return keyBytes, nil
}

func (s *IamService) ListServiceAccounts(googleProjectId string) ([]*iam.ServiceAccount, error) {
	var size int64 = 100
	accounts := make([]*iam.ServiceAccount, 0)
	resp, err := s.Service.Projects.ServiceAccounts.List("projects/" + googleProjectId).PageSize(size).Do()
	if err != nil {
		return nil, err
	}
	accounts = append(accounts, resp.Accounts...)
	nextPageToken := resp.NextPageToken
	for ; nextPageToken != ""; {
		r, e := s.Service.Projects.ServiceAccounts.List("projects/" + googleProjectId).PageSize(size).PageToken(nextPageToken).Do()
		if e != nil {
			return nil, e
		}
		nextPageToken = r.NextPageToken
		accounts = append(accounts, r.Accounts...)
	}
	return accounts, nil
}

func (s *IamService) RemoveExistingKeys(acc *iam.ServiceAccount) error {
	list, err := s.Service.Projects.ServiceAccounts.Keys.List("projects/-/serviceAccounts/" + acc.UniqueId).Do()
	if err != nil {
		log.Println("Fail to list service account keys")
		return err
	}
	if len(list.Keys) == 0 {
		log.Println("No account key to remove")
	}
	for _, key := range list.Keys {
		if key.KeyType == "SYSTEM_MANAGED" {
			continue
		}
		if _,err := s.Service.Projects.ServiceAccounts.Keys.Delete(key.Name).Do(); err != nil {
			log.Println("Fail to to remove key", key.Name)
		}
	}

	return nil
}
