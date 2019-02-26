package entity

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

type DriveAccount struct {
	Id                   bson.ObjectId `json:"id" bson:"_id"`
	AccountId            int64         `json:"accountId" bson:"accountId"`
	Name                 string        `json:"name" bson:"name"`
	Desc                 string        `json:"desc" bson:"desc"`
	Type                 string        `json:"type" bson:"type"`
	ClientEmail          string        `json:"clientEmail" bson:"clientEmail"`
	ClientId             string        `json:"clientId" bson:"clientId"`
	Key                  string        `json:"key" bson:"key"`
	Usage                int64         `json:"usage" bson:"usage"`
	Limit                int64         `json:"limit" bson:"limit"`
	Owner                bson.ObjectId `json:"owner" bson:"owner"`
	ProjectId            bson.ObjectId `json:"projectId" bson:"projectId"`
	QuotaUpdateTimestamp time.Time     `json:"quotaUpdateTimestamp" bson:"quotaUpdateTimestamp"`
}
