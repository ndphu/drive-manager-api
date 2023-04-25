package entity

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	Id primitive.ObjectID `json:"id" bson:"_id"`
	Email string `json:"email" bson:"email"`
	DisplayName string `json:"displayName" bson:"displayName"`
	Roles []string `json:"roles" bson:"roles"`
}
