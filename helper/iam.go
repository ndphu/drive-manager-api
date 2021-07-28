package helper

import (
	"context"
	"encoding/base64"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"log"
)

type IamService struct {
	Service *iam.Service
	Config *jwt.Config
}

func NewIamService(key []byte) (*IamService, error) {
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
