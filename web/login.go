package web

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lanthora/cucurbita/storage"
)

func LoginMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		route := c.Request.URL.String()
		if route == "/login" || route == "/register" || route == "/favicon.ico" {
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

func RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

func Login(c *gin.Context) {
	username := c.PostForm("username")
	password := sha256base64(c.PostForm("password"))
	if username == "" {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	currentUser := &User{}
	result := storage.Model(&User{}).Where("name = ?", username).Take(&currentUser)
	if result.Error != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	if currentUser.Password != password {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	currentUser.Token = uuid.New().String()
	storage.Save(currentUser)
	c.SetCookie("username", currentUser.Name, 86400, "/", "", false, false)
	c.SetCookie("token", currentUser.Token, 86400, "/", "", false, false)
	c.Redirect(http.StatusSeeOther, "/")
}

func sha256base64(input string) string {
	hash := sha256.Sum256([]byte(input))
	return base64.StdEncoding.EncodeToString(hash[:])
}
