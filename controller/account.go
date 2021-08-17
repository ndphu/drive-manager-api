package controller

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/ndphu/drive-manager-api/middleware"
	"github.com/ndphu/drive-manager-api/service"
	"google.golang.org/api/drive/v3"
	"log"
)

func AccountController(r *gin.RouterGroup) {
	accountService := service.GetAccountService()
	googleService := service.GoogleService{}
	r.Use(middleware.FirebaseAuthMiddleware())

	r.GET("/account/:id", func(c *gin.Context) {
		userId := CurrentUser(c).Id.Hex()
		accountId := c.Param("id")
		acc, err := accountService.FindAccountLookup(accountId, userId)
		if err != nil {
			status := 500
			if err == mgo.ErrNotFound {
				status = 404
			}
			c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
			return
		} else {
			c.JSON(200, gin.H{"success": true, "account": acc})
		}
	})

	r.GET("/account/:id/file/:fileId/download", func(c *gin.Context) {
		downloadDetails, err := googleService.GetDownloadLink(c.Param("id"), c.Param("fileId"))
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"success": true, "download": downloadDetails})
		}
	})

	r.POST("/account/:id/file/:fileId/favorite", func(c *gin.Context) {
		user := CurrentUser(c)
		userId := user.Id.Hex()
		accountId := c.Param("id")
		fileId := c.Param("fileId")
		if fv, err := accountService.SetFileFavorite(userId, accountId, fileId, true); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"success": true, "favorite": fv})
		}
	})

	r.POST("/account/:id/file/:fileId/sync", func(c *gin.Context) {
		user := CurrentUser(c)
		userId := user.Id.Hex()
		accountId := c.Param("id")
		var file drive.File
		if err := c.ShouldBindJSON(&file); err != nil {
			log.Println("Fail to parse file response...")
			c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
			return
		}

		if fv, err := accountService.SyncFile(userId, accountId, file); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		} else {
			if err := accountService.UpdateCachedQuotaByAccountIdAndAdditionalSize(accountId, file.Size); err != nil {
				c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			} else {
				c.JSON(200, gin.H{"success": true, "file": fv})
			}
		}
	})

	r.POST("/account/:id/file/:fileId/oldSync", func(c *gin.Context) {
		user := CurrentUser(c)
		userId := user.Id.Hex()
		accountId := c.Param("id")
		fileId := c.Param("fileId")

		if fv, err := accountService.SyncFileById(userId, accountId, fileId); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			if err := accountService.UpdateCachedQuotaByAccountId(accountId); err != nil {
				c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			} else {
				c.JSON(200, gin.H{"success": true, "file": fv})
			}
		}
	})

	r.GET("/account/:id/key", func(c *gin.Context) {
		account, err := accountService.FindAccount(c.Param("id"))
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}
		key := []byte(account.Key)
		c.String(200, base64.StdEncoding.EncodeToString(key))
	})

	r.POST("/account/:id/syncQuota", func(c *gin.Context) {
		//user := CurrentUser(c)
		//userId := user.Id.Hex()
		// TODO: permission!!!
		accountId := c.Param("id")
		if err := accountService.UpdateCachedQuotaByAccountId(accountId); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"success": true})
		}
	})
}
