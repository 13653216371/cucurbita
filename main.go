package main

import (
	"github.com/gin-gonic/gin"
	"github.com/lanthora/cucurbita/candy"
	"github.com/lanthora/cucurbita/logger"
	"github.com/lanthora/cucurbita/settings"
	"github.com/lanthora/cucurbita/web"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	r := gin.New()
	r.HTMLRender = web.HTMLRender
	r.Use(candy.WebsocketMiddleware(), web.LoginMiddleware())

	r.GET("/", web.Index)
	r.GET("/favicon.ico", web.Favicon)

	r.GET("/register", web.RegisterPage)
	r.POST("/register", web.UserRegister)

	r.GET("/login", web.LoginPage)
	r.POST("/login", web.Login)

	r.GET("/user", web.UserPage)
	r.GET("/user/delete", web.DeleteUser)

	r.GET("/domain", web.DomainPage)
	r.GET("/domain/insert", web.InsertDomainPage)
	r.POST("/domain/insert", web.InsertDomain)
	r.GET("/domain/delete", web.DeleteDomain)

	r.GET("/device", web.DevicePage)
	r.GET("/device/delete", web.DeleteDevice)

	r.GET("/logger", logger.SetLevel)

	if err := r.Run(settings.Address); err != nil {
		logger.Fatal(err)
	}
}
