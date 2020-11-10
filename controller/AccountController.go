package controller

import (
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"drive-manager-api/entity"
	"drive-manager-api/middleware"
	"drive-manager-api/service"
	"drive-manager-api/utils"
	helper "github.com/ndphu/google-api-helper"
	"net/http"
	"strconv"
	"strings"
)

func AccountController(r *gin.RouterGroup) error {
	accountService, err := service.GetAccountService()
	if err != nil {
		return err
	}
	r.Use(middleware.FirebaseAuthMiddleware())
	r.POST("", func(c *gin.Context) {
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

	r.GET("", func(c *gin.Context) {
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

	r.GET("/:id", func(c *gin.Context) {
		acc, err := accountService.FindAccount(c.Param("id"))
		if err != nil {
			ServerError("Fail to get account", err, c)
			return
		}
		c.JSON(200, gin.H{
			"id":   acc.Id,
			"name":  acc.Name,
			"desc":  acc.Desc,
			"limit": acc.Limit,
			"usage": acc.Usage,
		})
	})

	r.POST("/:id/key", func(c *gin.Context) {
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

	r.GET("/:id/files", func(c *gin.Context) {
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
		acc, err := accountService.FindAccount(c.Param("id"))
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
		c.JSON(200, files)
	})

	r.GET("/:id/file/:fileId/download", func(c *gin.Context) {
		acc, err := accountService.FindAccount(c.Param("id"))
		if err != nil {
			ServerError("Fail to get account", err, c)
			return
		}
		driveService, err := helper.GetDriveService([]byte(acc.Key))
		if err != nil {
			ServerError("Fail to initialize drive service", err, c)
			return
		}
		//driveFile, link, err := driveService.GetDownloadLink(c.Param("fileId"))

		driveFile, link, err := driveService.GetDownloadLink(c.Param("fileId"))
		if err != nil {
			ServerError("Fail to get download link", err, c)
			return
		}
		c.JSON(200, gin.H{"file": driveFile, "link": link,
		})
	})

	r.GET("/:id/file/:fileId/sharableLink", func(c *gin.Context) {
		acc, err := accountService.FindAccount(c.Param("id"))
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
		resp, err := http.Get(downloadLink)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		href := doc.Find("#uc-download-link").First().AttrOr("href", "")
		c.JSON(200, gin.H{"file": driveFile, "link": link,
			"directLink" : "https://drive.google.com" + strings.ReplaceAll(href, "&amp;", "&"),
			"cookie": resp.Header.Get("Set-Cookie")})
	})

	r.GET("/:id/refreshQuota", func(c *gin.Context) {
		driveAccount, err := accountService.FindAccount(c.Param("id"))
		if err != nil {
			ServerError("Fail to find account", err, c)
			return
		}
		if err := accountService.UpdateCachedQuota(driveAccount); err != nil {
			ServerError("Fail to get account", err, c)
			return
		}

		driveAccount.Key = ""

		c.JSON(200, driveAccount)
	})

	r.GET("/:id/key", func(c *gin.Context) {
		account, err := accountService.FindAccount(c.Param("id"))
		if err != nil {
			ServerError("Fail to get account", err, c)
			return
		}
		key := []byte(account.Key)
		c.String(200, base64.StdEncoding.EncodeToString(key))
	})

	r.POST("/:id/upload", func(c *gin.Context) {
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

	r.GET("/:id/accessToken", func(c *gin.Context) {
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

	return nil
}
