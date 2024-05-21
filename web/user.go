package web

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/foolin/goview"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lanthora/cucurbita/logger"
	"github.com/lanthora/cucurbita/storage"
)

type User struct {
	Name       string `gorm:"primaryKey"`
	Password   string
	Token      string
	Role       string
	Invitation string
}

func init() {
	err := storage.AutoMigrate(User{})
	if err != nil {
		logger.Fatal(err)
	}
}

func UserPage(c *gin.Context) {
	var users []User
	currentUser := c.MustGet("user").(*User)

	if currentUser.Role == "admin" {
		storage.Model(&User{}).Find(&users)

	}

	c.HTML(http.StatusOK, "user.html", goview.M{
		"users": users,
	})
}

func isValidInvitation(invitation string) bool {
	bytes, err := base64.RawStdEncoding.DecodeString(invitation)
	if err != nil {
		return false
	}
	info := strings.Split(string(bytes), "::")
	if len(info) != 2 {
		return false
	}
	if info[1] == "" {
		return false
	}

	count := int64(0)
	storage.Model(&User{}).Where("name = ? and invitation = ?", info[0], info[1]).Count(&count)
	return count == 1
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
		storage.Create(currentUser)
		c.SetCookie("username", currentUser.Name, 86400, "/", "", false, false)
		c.SetCookie("token", currentUser.Token, 86400, "/", "", false, false)
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	if !isValidInvitation(invitation) {
		c.Redirect(http.StatusSeeOther, "/register")
		return
	}

	storage.Model(&User{}).Where("name = ?", username).Count(&count)
	if count != 0 {
		c.Redirect(http.StatusSeeOther, "/register")
		return
	}

	currentUser.Name = username
	currentUser.Password = password
	currentUser.Token = uuid.New().String()
	currentUser.Role = "normal"
	storage.Create(currentUser)
	c.SetCookie("username", currentUser.Name, 86400, "/", "", false, false)
	c.SetCookie("token", currentUser.Token, 86400, "/", "", false, false)
	c.Redirect(http.StatusSeeOther, "/")
}

func DeleteUser(c *gin.Context) {
	currentUser := c.MustGet("user").(*User)
	if currentUser.Role == "admin" {
		storage.Delete(&User{Name: c.Query("name")})
	}
	c.Redirect(http.StatusSeeOther, c.GetHeader("Referer"))
}
