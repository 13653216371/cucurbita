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

		username, usernameErr := c.Cookie("username")
		token, tokenErr := c.Cookie("token")
		if usernameErr != nil || tokenErr != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}

		user := &User{Name: username}
		result := storage.Where(user).Take(user)
		if result.Error != nil || user.Token != token {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

func LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func Login(c *gin.Context) {
	currentUser := &User{}

	username := c.PostForm("username")
	password := sha256base64(c.PostForm("password"))
	if username == "" {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// 第一个注册的用户设置为管理员
	result := storage.Model(&User{}).Take(&currentUser)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		currentUser.Name = username
		currentUser.Password = password
		currentUser.Token = uuid.New().String()
		currentUser.Role = "admin"
		storage.Create(currentUser)
		c.SetCookie("username", currentUser.Name, 86400, "/", "", false, false)
		c.SetCookie("token", currentUser.Token, 86400, "/", "", false, false)
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// 后续注册的用户设置为普通用户
	result = storage.Model(&User{}).Where("name = ?", username).Take(&currentUser)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		currentUser.Name = username
		currentUser.Password = password
		currentUser.Token = uuid.New().String()
		currentUser.Role = "normal"
		storage.Create(currentUser)
		c.SetCookie("username", currentUser.Name, 86400, "/", "", false, false)
		c.SetCookie("token", currentUser.Token, 86400, "/", "", false, false)
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// 用户存在且密码匹配,登录成功,并更新 Token
	if currentUser.Password == password {
		currentUser.Token = uuid.New().String()
		storage.Save(currentUser)
		c.SetCookie("username", currentUser.Name, 86400, "/", "", false, false)
		c.SetCookie("token", currentUser.Token, 86400, "/", "", false, false)
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// 用户名密码不匹配,重新登录
	c.Redirect(http.StatusSeeOther, "/login")
}

func sha256base64(input string) string {
	hash := sha256.Sum256([]byte(input))
	return base64.StdEncoding.EncodeToString(hash[:])
}
