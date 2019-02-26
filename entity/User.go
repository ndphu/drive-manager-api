package entity

import "github.com/globalsign/mgo/bson"

type User struct {
	Id bson.ObjectId `json:"id" bson:"_id"`
	Email string `json:"email" bson:"email"`
	DisplayName string `json:"displayName" bson:"displayName"`
	Roles []string `json:"roles" bson:"roles"`
}
