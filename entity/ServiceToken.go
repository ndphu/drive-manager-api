package entity

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

type ServiceToken struct {
	Id        bson.ObjectId `json:"id" bson:"_id"`
	CreatedAt time.Time     `json:"createdAt" bson:"createdAt"`
	UserId    bson.ObjectId `json:"userId" bson:"userId"`
	Token     string        `json:"token" bson:"token"`
	TokenId   string        `json:"tokenId" bson:"tokenId"`
}
