package web

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lanthora/cucurbita/storage"
	"gorm.io/gorm"
)

func LoginMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		route := c.Request.URL.String()
		if route == "/login" || route == "/favicon.ico" {
			c.Next()
			return
		}

		token, err := c.Cookie("token")
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}

		config := &storage.Config{Key: "token"}
		result := storage.Where(config).Take(config)
		if result.Error != nil || config.Value != token {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func Login(c *gin.Context) {
	config := &storage.Config{Key: "password"}
	result := storage.Where(config).Take(config)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		config.Value = sha256base64(c.PostForm("password"))
		storage.Create(config)
	}

	if config.Value != sha256base64(c.PostForm("password")) {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	token := uuid.New().String()
	storage.Save(&storage.Config{Key: "token", Value: token})
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("token", token, 86400, "/", "", false, false)
	c.Redirect(http.StatusSeeOther, "/")
}

func sha256base64(input string) string {
	hash := sha256.Sum256([]byte(input))
	return base64.StdEncoding.EncodeToString(hash[:])
}
