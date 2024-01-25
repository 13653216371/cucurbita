package web

import (
	"embed"
	"path/filepath"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
)

//go:embed views/*
var views embed.FS

func fileHandler(config goview.Config, fileName string) (string, error) {
	path := filepath.Join(config.Root, fileName)
	bytes, err := views.ReadFile(path + config.Extension)
	return string(bytes), err
}

var HTMLRender *ginview.ViewEngine

func init() {
	HTMLRender = ginview.New(
		goview.Config{
			Root:      "views",
			Extension: ".html",
		},
	)
	HTMLRender.ViewEngine.SetFileHandler(fileHandler)
}
