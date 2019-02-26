package entity

import "github.com/globalsign/mgo/bson"

type Project struct {
	Id          bson.ObjectId `json:"id" bson:"_id"`
	DisplayName string        `json:"displayName" bson:"displayName"`
	Owner       bson.ObjectId `json:"owner" bson:"owner"`
	ProjectId   string        `json:"projectId" bson:"projectId"`
	AdminKey    string        `json:"adminKey" bson:"adminKey"`
}
