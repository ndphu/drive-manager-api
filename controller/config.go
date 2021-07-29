package controller

import (
	"drive-manager-api/dao"
	"github.com/gin-gonic/gin"
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
		var fc FirebaseConfig
		if err := dao.Collection("firebase_config").Find(nil).One(&fc); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"config": fc})
		}
	})
}
