package web

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"sort"
	"time"

	"github.com/foolin/goview"
	"github.com/gin-gonic/gin"
	"github.com/lanthora/cucurbita/candy"
	"github.com/lanthora/cucurbita/storage"
)

func DevicePage(c *gin.Context) {

	candy.Sync()

	var devices []candy.Device

	switch c.Query("active") {
	case "online":
		storage.Model(&candy.Device{}).Where("online = true").Find(&devices)
	case "daily":
		storage.Model(&candy.Device{}).Where("online = true").Or("conn_updated_at > ?", time.Now().AddDate(0, 0, -1)).Find(&devices)
	case "weekly":
		storage.Model(&candy.Device{}).Where("online = true").Or("conn_updated_at > ?", time.Now().AddDate(0, 0, -7)).Find(&devices)
	case "dormant":
		storage.Model(&candy.Device{}).Where("online = false AND conn_updated_at < ?", time.Now().AddDate(0, 0, -7)).Find(&devices)
	default:
		storage.Find(&devices)
	}

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].Domain == devices[j].Domain {
			a := net.ParseIP(devices[i].IP)
			b := net.ParseIP(devices[j].IP)
			return bytes.Compare(a, b) < 0
		}
		return devices[i].Domain < devices[j].Domain
	})

	c.HTML(http.StatusOK, "device.html", goview.M{
		"devices": devices,
		"formatRxTx": func(n uint64) string {
			size := float64(n)
			units := []string{"B", "KB", "MB", "GB", "TB", "EB", "PB"}
			idx := 0
			for size > 1024 {
				size = size / 1024
				idx++
			}
			return fmt.Sprintf("%.2f %v", size, units[idx])
		},
	})
}

func DeleteDevice(c *gin.Context) {
	storage.Delete(&candy.Device{Domain: c.Query("domain"), VMac: c.Query("vmac")})
	c.Redirect(http.StatusSeeOther, c.GetHeader("Referer"))
}
