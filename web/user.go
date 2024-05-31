package web

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/foolin/goview"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lanthora/cucurbita/candy"
	"github.com/lanthora/cucurbita/logger"
	"github.com/lanthora/cucurbita/storage"
)

type User struct {
	Name       string `gorm:"primaryKey"`
	Password   string
	Token      string
	Role       string
	Invitation uint32
	Inviter    string
}

func init() {
	err := storage.AutoMigrate(User{})
	if err != nil {
		logger.Fatal(err)
	}
}

func RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

func UserRegister(c *gin.Context) {
	username := c.PostForm("username")
	password := sha256base64(c.PostForm("password"))
	invitation := c.PostForm("invitation")

	count := int64(0)
	currentUser := &User{}
	storage.Model(&User{}).Count(&count)
	if count == 0 {
		currentUser.Name = username
		currentUser.Password = password
		currentUser.Token = uuid.New().String()
		currentUser.Role = "admin"
		currentUser.Invitation = rand.Uint32()
		currentUser.Inviter = "-"
		storage.Create(currentUser)
		c.SetCookie("username", currentUser.Name, 86400, "/", "", false, false)
		c.SetCookie("token", currentUser.Token, 86400, "/", "", false, false)
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	storage.Model(&User{}).Where("name = ?", username).Count(&count)
	if count != 0 {
		c.Redirect(http.StatusSeeOther, "/register")
		return
	}

	inviter, ok := isValidInvitation(invitation)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/register")
		return
	}

	currentUser.Name = username
	currentUser.Password = password
	currentUser.Token = uuid.New().String()
	currentUser.Role = "normal"
	currentUser.Invitation = rand.Uint32()
	currentUser.Inviter = inviter
	storage.Create(currentUser)
	c.SetCookie("username", currentUser.Name, 86400, "/", "", false, false)
	c.SetCookie("token", currentUser.Token, 86400, "/", "", false, false)
	c.Redirect(http.StatusSeeOther, "/")
}

func isValidInvitation(invitation string) (inviter string, ok bool) {
	bytes, err := base64.RawStdEncoding.DecodeString(invitation)
	if err != nil {
		return
	}
	info := strings.Split(string(bytes), "::")
	if len(info) != 2 {
		return
	}
	if info[1] == "" {
		return
	}

	user := &User{}
	result := storage.Model(&User{}).Where("name = ? and invitation = ?", info[0], info[1]).Take(&user)
	if result.Error != nil {
		return
	}

	inviter = info[0]
	ok = true

	user.Invitation = rand.Uint32()
	storage.Save(user)
	return
}

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

func UserPage(c *gin.Context) {
	var users []User
	currentUser := c.MustGet("user").(*User)

	if currentUser.Role == "admin" {
		storage.Model(&User{}).Find(&users)
	} else {
		storage.Model(&User{}).Where("name = ?", currentUser.Name).Or("inviter = ?", currentUser.Name).Find(&users)
	}

	c.HTML(http.StatusOK, "user.html", goview.M{
		"users": users,
		"encodeInvitation": func(name string, invitation uint64) string {
			return base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf("%s::%d", name, invitation)))
		},
	})
}

func DeleteUser(c *gin.Context) {
	currentUser := c.MustGet("user").(*User)
	user := &User{}
	result := storage.Model(&User{}).Where("name = ?", c.Query("name")).Take(user)
	if result.Error != nil {
		c.Redirect(http.StatusSeeOther, c.GetHeader("Referer"))
		return
	}
	domainCount := int64(0)
	storage.Model(&candy.Domain{}).Where("username = ?", user.Name).Limit(1).Count(&domainCount)
	if domainCount != 0 {
		c.Redirect(http.StatusSeeOther, c.GetHeader("Referer"))
		return
	}
	devCount := int64(0)
	storage.Model(&candy.Device{}).Where("username = ?", user.Name).Limit(1).Count(&devCount)
	if devCount != 0 {
		c.Redirect(http.StatusSeeOther, c.GetHeader("Referer"))
		return
	}
	if (currentUser.Role == "admin") != (user.Name == currentUser.Name) {
		storage.Model(&User{}).Where("inviter = ?", user.Name).Update("inviter", user.Inviter)
		storage.Delete(user)
	}
	c.Redirect(http.StatusSeeOther, c.GetHeader("Referer"))
}
