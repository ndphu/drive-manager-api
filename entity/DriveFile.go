package entity

import "go.mongodb.org/mongo-driver/bson/primitive"

type DriveFile struct {
	Id           primitive.ObjectID `json:"id" bson:"_id"`
	Quality      string        `json:"quality" bson:"quality"`
	Name         string        `json:"name" bson:"name"`
	Size         int64         `json:"size" bson:"size"`
	DriveFileId  string        `json:"driveId" bson:"driveId"`
	DriveAccount primitive.ObjectID `json:"driveAccount" bson:"driveAccount"`
}
