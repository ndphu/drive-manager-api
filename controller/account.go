package controller

import (
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/helper"
	"github.com/ndphu/drive-manager-api/middleware"
	"github.com/ndphu/drive-manager-api/service"
	"github.com/ndphu/drive-manager-api/utils"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"strconv"
)

func AccountController(r *gin.RouterGroup) {
	accountService := service.GetAccountService()
	googleService := service.GoogleService{}
	r.Use(middleware.FirebaseAuthMiddleware())
	r.POST("/accounts", func(c *gin.Context) {
		da := entity.DriveAccount{}
		err := c.BindJSON(&da)
		if err != nil {
			BadRequest("Fail to parse body", err, c)
			return
		}

		keyDecoded, err := base64.StdEncoding.DecodeString(da.Key)
		if err != nil {
			BadRequest("Fail to decode base64 key data", err, c)
			return
		}

		driveService, err := helper.GetDriveService([]byte(keyDecoded))
		if err != nil {
			BadRequest("Invalid JSON Key", err, c)
			return
		}

		if _, err := driveService.GetQuotaUsage(); err != nil {
			BadRequest("Invalid JSON Key", err, c)
			return
		}

		accountService.InitializeKey(&da, keyDecoded)
		if err != nil {
			ServerError("Fail to key data", err, c)
			return
		}

		err = accountService.Save(&da)
		if err != nil {
			ServerError("Fail to save drive account", err, c)
			return
		}
		c.JSON(200, da)
	})

	r.GET("/accounts", func(c *gin.Context) {
		page := utils.GetIntQuery(c, "page", 1)
		size := utils.GetIntQuery(c, "size", 10)
		val, _ := c.Get("user")
		user := val.(*entity.User)
		accList, hasMore, err := accountService.FindAccounts(page, size, false, user.Id.Hex())
		if err != nil {
			ServerError("Fail to get account list", err, c)
			return
		}
		if accList == nil {
			accList = []*entity.DriveAccount{}
		}

		c.JSON(200, gin.H{
			"accounts": accList,
			"hasMore":  hasMore,
			"page":     page,
			"size":     size,
		})
	})

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

	r.POST("/account/:id/key", func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil {
			BadRequest("Request required body as base64", err, c)
			return
		}
		keyDecoded, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			BadRequest("Fail to decode base64 key data", err, c)
			return
		}
		err = accountService.UpdateKey(c.Param("id"), keyDecoded)
		if err != nil {
			ServerError("Fail to initialize key for account", err, c)
			return
		}

		account, err := accountService.FindAccount(c.Param("id"))
		if err != nil {
			ServerError("Fail to query account", err, c)
			return
		}
		c.JSON(200, account)
	})

	r.GET("/account/:id/files", func(c *gin.Context) {
		accountId := c.Param("id")

		page, err := strconv.Atoi(c.Query("page"))
		if err != nil {
			BadRequest("Invalid page parameter", err, c)
			return
		}
		size, err := strconv.Atoi(c.Query("size"))
		if err != nil {
			BadRequest("Invalid size parameter", err, c)
			return
		}
		acc, err := accountService.FindAccount(accountId)
		if err != nil {
			ServerError("Fail to get account", err, c)
			return
		}
		driveService, err := helper.GetDriveService([]byte(acc.Key))
		if err != nil {
			ServerError("Fail to initialize drive service", err, c)
			return
		}
		files, err := driveService.ListFiles(page, int64(size))
		if err != nil {
			ServerError("Fail to list file", err, c)
			return
		}
		// TODO
		for _, f := range files {
			f.AccountId = accountId
		}
		c.JSON(200, files)
	})

	r.GET("/account/:id/file/:fileId/download", func(c *gin.Context) {
		downloadDetails, err := googleService.GetDownloadLink(c.Param("id"), c.Param("fileId"))
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"success": true, "download": downloadDetails})
		}
	})

	r.GET("/account/:id/file/:fileId/sharableLink", func(c *gin.Context) {
		accountId := c.Param("id")
		acc, err := accountService.FindAccount(accountId)
		if err != nil {
			ServerError("Fail to get account", err, c)
			return
		}
		driveService, err := helper.GetDriveService([]byte(acc.Key))
		if err != nil {
			ServerError("Fail to initialize drive service", err, c)
			return
		}
		driveFile, link, err := driveService.GetSharableLink(c.Param("fileId"))
		if err != nil {
			ServerError("Fail to get download link", err, c)
			return
		}
		downloadLink := fmt.Sprintf("https://drive.google.com/uc?id=%s&export=download", c.Param("fileId"))

		c.JSON(200, gin.H{
			"file":       driveFile,
			"link":       link,
			"directLink": downloadLink,
		})
	})

	r.GET("/account/:id/refreshQuota", func(c *gin.Context) {
		driveAccount, err := accountService.FindAccount(c.Param("id"))
		if err != nil {
			ServerError("Fail to find account", err, c)
			return
		}
		if err := accountService.UpdateCachedQuota(driveAccount); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			ServerError("Fail to get account", err, c)
			return
		}

		driveAccount.Key = ""

		c.JSON(200, driveAccount)
	})

	r.GET("/account/:id/key", func(c *gin.Context) {
		account, err := accountService.FindAccount(c.Param("id"))
		if err != nil {
			ServerError("Fail to get account", err, c)
			return
		}
		key := []byte(account.Key)
		c.String(200, base64.StdEncoding.EncodeToString(key))
	})

	r.POST("/account/:id/upload", func(c *gin.Context) {
		user := CurrentUser(c)
		account, err := accountService.FindAccount(c.Param("id"))
		if err != nil || account.Owner.Hex() != user.Id.Hex() {
			ServerError("Account not found", err, c)
			return
		}
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			ServerError("Cannot read uploaded file", err, c)
			return
		}

		srv, err := helper.GetDriveService([]byte(account.Key))
		if err != nil {
			ServerError("Fail to load drive account", err, c)
			return
		}

		uploadedFile, err := srv.UploadFileFromStream(header.Filename, header.Filename, "", file)
		if err != nil {
			ServerError("Fail to upload", err, c)
			return
		}

		c.JSON(200, uploadedFile)
	})

	r.GET("/account/:id/accessToken", func(c *gin.Context) {
		account, err := accountService.FindAccountById(bson.ObjectIdHex(c.Param("id")), CurrentUser(c).Id)
		if err != nil {
			ServerError("Fail to get account", err, c)
			return
		}
		token, err := accountService.GetAccessToken(account)
		if err != nil {
			ServerError("Fail to get access token", err, c)
			return
		}
		c.JSON(200, gin.H{"accessToken": token})
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
		fileId := c.Param("fileId")
		if fv, err := accountService.SyncFile(userId, accountId, fileId); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{"success": true, "file": fv})
		}
	})
}
