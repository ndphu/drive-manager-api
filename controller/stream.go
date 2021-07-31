package controller

import (
	"drive-manager-api/service"
	"github.com/gin-gonic/gin"
)

func StreamController(r *gin.RouterGroup) error {
	redisService, err := service.GetRedisService()
	if err != nil {
		panic(err)
	}
	r.GET("/:id", func(c *gin.Context) {
		fileId := c.Param("id")
		authCookie, err := redisService.Get("file:" + fileId + ":auth")
		if err != nil || authCookie == "" {
			c.JSON(500, gin.H{"error": "stream " + fileId + " not found"})
			return
		}
		url, err := redisService.Get("file:" + fileId + ":url")
		if err != nil || url == "" {
			c.JSON(500, gin.H{"error": "stream " + fileId + " not found"})
			return
		}

		//stream(c, url, authCookie)
	})

	return nil
}
