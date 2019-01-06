package entity

import "github.com/globalsign/mgo/bson"

type DriveAccount struct {
	Id bson.ObjectId `json:"_id" bson:"_id"`
	AccountId int64 `json:"accountId" bson:"accountId"`
	Name string `json:"name" bson:"name"`
	Desc string `json:"desc" bson:"desc"`
	Type string `json:"type" bson:"type"`
	ProjectId string `json:"projectId" bson:"projectId"`
	ClientEmail string `json:"clientEmail" bson:"clientEmail"`
	ClientId string `json:"clientId" bson:"clientId"`
	Key string `json:"key" bson:"key"`
	Usage int64 `json:"usage" bson:"usage"`
	Limit int64 `json:"limit" bson:"limit"`
}

