package web

import (
	"net/http"
	"time"

	"github.com/foolin/goview"
	"github.com/gin-gonic/gin"
	"github.com/lanthora/cucurbita/candy"
	"github.com/lanthora/cucurbita/logger"
	"github.com/lanthora/cucurbita/storage"
)

type User struct {
	Name      string `gorm:"primaryKey"`
	Password  string
	Token     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
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

func DeleteUser(c *gin.Context) {
	currentUser := c.MustGet("user").(*User)
	if currentUser.Role == "admin" || candy.GetDomain(c.Query("name")).Username == currentUser.Name {
		storage.Delete(&User{Name: c.Query("name")})
	}
	c.Redirect(http.StatusSeeOther, c.GetHeader("Referer"))
}
