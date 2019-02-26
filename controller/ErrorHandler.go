package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
)

func BadRequest(msg string, err error, c *gin.Context) {
	c.AbortWithStatusJSON(400, gin.H{
		"code": 400,
		"err":  fmt.Sprintf("%v", err),
		"msg":  msg,
	})
	c.Abort()
}

func ServerError(msg string, err error, c *gin.Context) {
	log.Printf("%s %v\n", msg, err)
	c.JSON(500, gin.H{
		"code": 500,
		"err":  fmt.Sprintf("%v", err),
		"msg":  msg,
	})
	c.Abort()
	//log.Fatalf("Internal Server Error: %s %v", msg, err)
}
