package candy

import (
	"net"

	"github.com/ip2location/ip2location-go/v9"
	"github.com/ipinfo/go/v2/ipinfo"
	"github.com/ipinfo/go/v2/ipinfo/cache"
	"github.com/lanthora/cucurbita/storage"
)

type dummyCacheEngine struct {
	cache map[string]interface{}
}

func newDummyCacheEngine() *dummyCacheEngine {
	return &dummyCacheEngine{
		cache: make(map[string]interface{}),
	}
}

func (c *dummyCacheEngine) Get(key string) (interface{}, error) {
	if v, ok := c.cache[key]; ok {
		return v, nil
	}
	return nil, cache.ErrNotFound
}

func (c *dummyCacheEngine) Set(key string, value interface{}) error {
	c.cache[key] = value
	return nil
}

var dummyCache = ipinfo.NewCache(newDummyCacheEngine())

func ip2CountryRegion(ip string) (country, region string) {
	config := &storage.Config{Key: "ipinfo"}
	if result := storage.Where(config).Take(config); result.Error == nil {
		client := ipinfo.NewClient(nil, dummyCache, config.Value)
		if info, err := client.GetIPInfo(net.ParseIP(ip)); err == nil {
			country = info.Country
			region = info.Region
			return
		}
	}

	if db, err := ip2location.OpenDB("/var/lib/cucurbita/IP2LOCATION.BIN"); err == nil {
		defer db.Close()
		if results, err := db.Get_all(ip); err == nil {
			country = results.Country_short
			region = results.Region
			return
		}
	}

	return
}

func UpdateLocation(device *Device, ip string) {
	device.Country, device.Region = ip2CountryRegion(ip)
	storage.Save(device)
}
