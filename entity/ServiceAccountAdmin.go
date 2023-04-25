package entity

import "go.mongodb.org/mongo-driver/bson/primitive"

type ServiceAccountAdmin struct{
	Id primitive.ObjectID `json:"id" bson:"_id"`
	UserId primitive.ObjectID `json:"userId" bson:"userId"`
	Key string `json:"key" bson:"key"`
}

