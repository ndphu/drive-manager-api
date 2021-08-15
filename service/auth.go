package service

import (
	"context"
	"encoding/base64"
	"firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/globalsign/mgo/bson"
	"github.com/golang-jwt/jwt"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/nu7hatch/gouuid"
	"google.golang.org/api/option"
	"log"
	"os"
	"time"
)

type AuthService struct {
	App         *firebase.App
	TokenSecret []byte
}

type FirebaseAccount struct {
	Id   bson.ObjectId `json:"id" bson:"_id"`
	Name string        `json:"name" bson:"name"`
	Key  string        `json:"key" bson:"key"`
}

var authService *AuthService

func GetAuthService() (*AuthService, error) {
	tokenSecret := os.Getenv("TOKEN_SECRET")
	if tokenSecret == "" {
		//return nil, errors.New("NoTokenSecret")
		panic("No Token Secret")
	}

	if authService == nil {
		adminAccount := FirebaseAccount{}
		err := dao.FirebaseAdmin().FindOne(nil, &adminAccount)
		if err != nil {
			log.Fatal("fail to get Firebase Admin key")
		}

		rawKey, err := base64.StdEncoding.DecodeString(adminAccount.Key)
		if err != nil {
			log.Fatal("fail to parse admin key")
		}

		opt := option.WithCredentialsJSON(rawKey)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			log.Fatalf("error initializing app: %v\n", err)
		}

		authService = &AuthService{
			App:         app,
			TokenSecret: []byte(tokenSecret),
		}
	}
	return authService, nil
}

func (s *AuthService) getAuthClient() (*auth.Client, error) {
	return s.App.Auth(context.Background())
}

func (s *AuthService) GetUserFromToken(jwtToken string) (*entity.User, error) {
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		mapClaims := token.Claims.(jwt.MapClaims)
		delete(mapClaims, "iat")
		return s.TokenSecret, nil
	})
	if err != nil {
		log.Println("Fail to parse jwt token by error:", err.Error())
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		roles := make([]string, 0)
		claimRoles, exist := claims["roles"]
		if exist {
			_roles := claimRoles.([]interface{})
			for _, role := range _roles {
				roles = append(roles, role.(string))
			}
		}

		return &entity.User{
			Id:    bson.ObjectIdHex(claims["user_id"].(string)),
			Email: claims["user_email"].(string),
			Roles: roles,
		}, nil
	} else {
		log.Println("fail to parse token")
		return nil, err
	}
}

func (s *AuthService) CreateUserWithEmail(email string, password string, displayName string) (*entity.User, error) {
	params := (&auth.UserToCreate{}).
		Email(email).
		EmailVerified(false).
		Password(password).
		DisplayName(displayName).
		Disabled(false)

	client, err := s.getAuthClient()
	if err != nil {
		return nil, err
	}

	u, err := client.CreateUser(context.Background(), params)
	if err != nil {
		log.Printf("error creating user: %v\n", err)
		return nil, err
	}

	log.Printf("successfully created user: %s\n", u.Email)
	user := entity.User{
		Id:          bson.NewObjectId(),
		DisplayName: displayName,
		Email:       u.Email,
		Roles:       []string{"user"},
	}
	if err := dao.User().Insert(&user); err != nil {
		log.Println("Fail to insert user by error", err.Error())
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) LoginWithFirebaseToken(firebaseToken string) (*entity.User, string, error) {
	client, err := s.App.Auth(context.Background())
	if err != nil {
		return nil, "", err
	}
	token, err := client.VerifyIDToken(context.Background(), firebaseToken)
	if err != nil {
		log.Println("Fail to parse token")
		return nil, "", err
	}
	var user entity.User
	if err := dao.User().FindOne(bson.M{
		"email": token.Claims["email"].(string),
	}, &user); err != nil {
		log.Println("Fai to find user in database by error", err.Error())
		return nil, "", err
	}

	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat":          now.Unix(),
		"exp":          now.AddDate(0, 0, 1).Unix(),
		"user_id":      user.Id.Hex(),
		"user_email":   user.Email,
		"display_name": user.DisplayName,
		"roles":        user.Roles,
		"provider":     "Firebase",
		"type":         "login_token",
	})
	jwtTokenString, err := jwtToken.SignedString(s.TokenSecret)
	return &user, jwtTokenString, err
}

func (s *AuthService) NewServiceToken(user *entity.User) (*entity.ServiceToken, error) {
	tokenId, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat":          now.Unix(),
		"exp":          now.AddDate(1, 0, 0).Unix(),
		"user_id":      user.Id.Hex(),
		"user_email":   user.Email,
		"display_name": user.DisplayName,
		"type":         "service_token",
		"token_id":     tokenId.String(),
	})
	token, err := jwtToken.SignedString(s.TokenSecret)
	if err != nil {
		return nil, err
	}

	st := entity.ServiceToken{
		Id:        bson.NewObjectId(),
		UserId:    user.Id,
		Token:     token,
		CreatedAt: now,
		TokenId:   tokenId.String(),
	}

	if err := dao.ServiceToken().Insert(&st); err != nil {
		log.Println("Fail to insert service_token by error", err.Error())
		return nil, err
	}

	return &st, err
}
