package controller

import (
	"drive-manager-api/service"
	"github.com/gin-gonic/gin"
)

func SyncController(r *gin.RouterGroup) {
	s := service.ProjectService{}

	r.POST("/project/:id", func(c *gin.Context) {
		user := CurrentUser(c)
		projectId := c.Param("id")
		if err := s.SyncProject(projectId, user.Id.Hex()); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err})
		} else {
			c.JSON(200, gin.H{"success": true})
		}
	})
}
