package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/ndphu/drive-manager-api/dao"
	"go.mongodb.org/mongo-driver/bson"
)

type FirebaseConfig struct {
	ApiKey            string `json:"apiKey" bson:"apiKey"`
	AuthDomain        string `json:"authDomain" bson:"authDomain"`
	DatabaseURL       string `json:"databaseURL" bson:"databaseURL"`
	ProjectId         string `json:"projectId" bson:"projectId"`
	StorageBucket     string `json:"storageBucket" bson:"storageBucket"`
	MessagingSenderId string `json:"messagingSenderId" bson:"MessagingSenderId"`
}

func ConfigController(r *gin.RouterGroup) {
	r.GET("/firebase", func(c *gin.Context) {
		/*var fc service.FirebaseAccount
		if err := dao.FirebaseAdmin().FindOne(bson.D{}, &fc); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"config": fc})
		}*/
		fc := FirebaseConfig{}
		if err := dao.RawCollection("firebase_config").FindOne(context.Background(), bson.D{}).Decode(&fc); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{
				"config": fc,
			})
		}
	})
}
