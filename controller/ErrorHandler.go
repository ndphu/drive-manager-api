package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func BadRequest(msg string, err error, c*gin.Context)  {
	c.JSON(400, gin.H{
		"code": 400,
		"err": fmt.Sprintf("%v", err),
		"msg": msg,
	})
	//log.Fatalf("Bad Request: %s %v", msg, err)
}

func ServerError(msg string, err error, c*gin.Context)  {
	c.JSON(500, gin.H{
		"code": 500,
		"err": fmt.Sprintf("%v", err),
		"msg": msg,
	})
	//log.Fatalf("Internal Server Error: %s %v", msg, err)
}