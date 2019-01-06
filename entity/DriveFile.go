package entity

import "github.com/globalsign/mgo/bson"

type DriveFile struct {
	Id           bson.ObjectId `json:"id" bson:"_id"`
	Quality      string        `json:"quality" bson:"quality"`
	Name         string        `json:"name" bson:"name"`
	Size         int64         `json:"size" bson:"size"`
	DriveFileId  string        `json:"driveId" bson:"driveId"`
	DriveAccount bson.ObjectId `json:"driveAccount" bson:"driveAccount"`
}
