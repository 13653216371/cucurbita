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
	storage.Find(&domains)

	c.HTML(http.StatusOK, "domain.html", goview.M{
		"domains": domains,
	})
}

func InsertDomainPage(c *gin.Context) {
	c.HTML(http.StatusOK, "domain/insert.html", nil)
}

func InsertDomain(c *gin.Context) {
	result := storage.Create(&candy.Domain{Name: c.PostForm("name"), Password: c.PostForm("password"), DHCP: c.PostForm("dhcp"), Broadcast: c.PostForm("broadcast") == "enable"})
	if result.Error != nil {
		c.Redirect(http.StatusSeeOther, "/domain/insert")
	} else {
		c.Redirect(http.StatusSeeOther, "/domain")
	}
}

func DeleteDomain(c *gin.Context) {
	candy.DeleteDomain(c.Query("name"))
	c.Redirect(http.StatusSeeOther, c.GetHeader("Referer"))
}
