package entity

import "github.com/globalsign/mgo/bson"

type ServiceAccountAdmin struct{
	Id bson.ObjectId `json:"id" bson:"_id"`
	UserId bson.ObjectId `json:"userId" bson:"userId"`
	Key string `json:"key" bson:"key"`
}

