package entity

import "github.com/globalsign/mgo/bson"

type Project struct {
	Id          primitive.ObjectID `json:"id" bson:"_id"`
	DisplayName string        `json:"displayName" bson:"displayName"`
	Owner       primitive.ObjectID `json:"owner" bson:"owner"`
	ProjectId   string        `json:"projectId" bson:"projectId"`
	AdminKey   string        `json:"adminKey,omitempty" bson:"adminKey,omitempty"`
}
