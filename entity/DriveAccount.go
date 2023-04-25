package entity

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

type DriveAccount struct {
	Id                   primitive.ObjectID `json:"id" bson:"_id"`
	Name                 string        `json:"name" bson:"name"`
	Desc                 string        `json:"desc" bson:"desc"`
	Type                 string        `json:"type" bson:"type"`
	ClientEmail          string        `json:"clientEmail" bson:"clientEmail"`
	ClientId             string        `json:"clientId" bson:"clientId"`
	Key                  string        `json:"key,omitempty" bson:"key"`
	Usage                int64         `json:"usage" bson:"usage"`
	Available            int64         `json:"available" bson:"available"`
	Limit                int64         `json:"limit" bson:"limit"`
	Owner                primitive.ObjectID `json:"owner" bson:"owner"`
	ProjectId            primitive.ObjectID `json:"projectId" bson:"projectId"`
	QuotaUpdateTimestamp time.Time     `json:"quotaUpdateTimestamp" bson:"quotaUpdateTimestamp"`
}
