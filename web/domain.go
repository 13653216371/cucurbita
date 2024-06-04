package web

import (
	"net/http"

	"github.com/foolin/goview"
	"github.com/gin-gonic/gin"
	"github.com/lanthora/cucurbita/candy"
	"github.com/lanthora/cucurbita/storage"
)

func DomainPage(c *gin.Context) {
	var domains []candy.Domain
	currentUser := c.MustGet("user").(*User)
	if currentUser.Role == "admin" {
		storage.Model(&candy.Domain{}).Find(&domains)
	} else {
		storage.Model(&candy.Domain{}).Where("username = ?", currentUser.Name).Find(&domains)
	}

	c.HTML(http.StatusOK, "domain.html", goview.M{
		"domains": domains,
	})
}

func InsertDomainPage(c *gin.Context) {
	c.HTML(http.StatusOK, "domain/insert.html", nil)
}

func InsertDomain(c *gin.Context) {
	currentUser := c.MustGet("user").(*User)
	result := storage.Create(&candy.Domain{
		Name:      c.PostForm("name"),
		Password:  c.PostForm("password"),
		DHCP:      c.PostForm("dhcp"),
		Broadcast: c.PostForm("broadcast") == "enable",
		Username:  currentUser.Name})
	if result.Error != nil {
		c.Redirect(http.StatusFound, "/domain/insert")
	} else {
		c.Redirect(http.StatusFound, "/domain")
	}
}

func DeleteDomain(c *gin.Context) {
	currentUser := c.MustGet("user").(*User)
	if currentUser.Role == "admin" || candy.GetDomain(c.Query("name")).Username == currentUser.Name {
		candy.DeleteDomain(c.Query("name"))
	}
	c.Redirect(http.StatusFound, c.GetHeader("Referer"))
}
