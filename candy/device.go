package candy

import (
	"encoding/binary"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lanthora/cucurbita/logger"
	"github.com/lanthora/cucurbita/storage"
)

func init() {
	err := storage.AutoMigrate(Domain{})
	if err != nil {
		logger.Fatal(err)
	}
}

type Device struct {
	Domain        string `gorm:"primaryKey"`
	VMac          string `gorm:"primaryKey"`
	IP            string
	Country       string
	Region        string
	Online        bool
	ConnUpdatedAt time.Time
	RX            uint64
	TX            uint64
	OS            string
	Version       string
	Hostname      string
	Username      string

	ip uint32
}

type Domain struct {
	Name      string `gorm:"primaryKey"`
	Password  string
	DHCP      string
	Broadcast bool
	Username  string

	mask   uint32
	netID  uint32
	hostID uint32

	mutex       sync.RWMutex
	wsDeviceMap map[*Websocket]*Device
	ipWsMap     map[uint32]*Websocket
}

type Websocket struct {
	conn  *websocket.Conn
	mutex sync.Mutex

	clientException bool
}

func (ws *Websocket) UpdateReadDeadline() error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.conn.SetReadDeadline((time.Now().Add(60 * time.Second)))
}

func (ws *Websocket) WriteMessage(buffer []byte) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.conn.WriteMessage(websocket.BinaryMessage, buffer)
}

func (ws *Websocket) WritePong(buffer []byte) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.conn.WriteMessage(websocket.PongMessage, buffer)
}

var nameDomainMap map[string]*Domain = make(map[string]*Domain)
var nameDomainMapMutex sync.RWMutex

func GetDomain(name string) *Domain {
	nameDomainMapMutex.Lock()
	defer nameDomainMapMutex.Unlock()

	domain, ok := nameDomainMap[name]
	if ok {
		return domain
	}

	result := storage.Model(&Domain{}).Where("name = ?", name).Take(&domain)
	if result.Error != nil {
		return nil
	}

	_, ipNet, err := net.ParseCIDR(domain.DHCP)
	if err == nil {
		domain.netID = binary.BigEndian.Uint32(ipNet.IP)
		domain.mask = binary.BigEndian.Uint32(ipNet.Mask)
		domain.hostID = rand.Uint32() & ^domain.mask

		if ^domain.mask < 2 {
			return nil
		}
		updateHostID(domain)
	}

	domain.wsDeviceMap = make(map[*Websocket]*Device)
	domain.ipWsMap = make(map[uint32]*Websocket)

	nameDomainMap[name] = domain
	return domain
}

func DeleteDomain(name string) {
	nameDomainMapMutex.Lock()
	defer nameDomainMapMutex.Unlock()

	if domain, ok := nameDomainMap[name]; ok {
		domain.mutex.RLock()
		defer domain.mutex.RUnlock()

		for ws := range domain.wsDeviceMap {
			ws.conn.Close()
		}
	}

	delete(nameDomainMap, name)
	storage.Delete(&Domain{Name: name})
}

func updateHostID(domain *Domain) {
	for ok := true; ok; ok = (domain.hostID == 0 || domain.hostID == ^domain.mask) {
		domain.hostID = (domain.hostID + 1) & (^domain.mask)
	}
}

func Sync() {
	nameDomainMapMutex.RLock()
	defer nameDomainMapMutex.RUnlock()

	for _, domain := range nameDomainMap {
		domain.mutex.RLock()
		defer domain.mutex.RUnlock()

		for _, device := range domain.wsDeviceMap {
			if device.Online {
				storage.Save(device)
			}
		}
	}
}
