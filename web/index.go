package web

import (
	"net/http"
	"time"

	"github.com/foolin/goview"
	"github.com/gin-gonic/gin"
	"github.com/lanthora/cucurbita/candy"
	"github.com/lanthora/cucurbita/storage"
)

func Index(c *gin.Context) {
	var domains []candy.Device
	storage.Find(&domains)

	online := int64(0)
	daily := int64(0)
	weekly := int64(0)
	domain := int64(0)

	storage.Model(&candy.Device{}).Where("online = true").Count(&online)
	storage.Model(&candy.Device{}).Where("online = true").Or("conn_updated_at > ?", time.Now().AddDate(0, 0, -1)).Count(&daily)
	storage.Model(&candy.Device{}).Where("online = true").Or("conn_updated_at > ?", time.Now().AddDate(0, 0, -7)).Count(&weekly)
	storage.Model(&candy.Domain{}).Count(&domain)

	c.HTML(http.StatusOK, "index.html", goview.M{
		"online": online,
		"daily":  daily,
		"weekly": weekly,
		"domain": domain,
	})
}

func Favicon(c *gin.Context) {
	buffer, err := views.ReadFile("views/favicon.ico")
	if err != nil {
		c.Status(http.StatusNotFound)
	} else {
		c.Data(http.StatusOK, "image/x-icon", buffer)
	}
}
