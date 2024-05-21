package candy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lanthora/cucurbita/logger"
	"github.com/lanthora/cucurbita/storage"
	"github.com/lunixbochs/struc"
	"gorm.io/gorm"
)

func init() {
	err := storage.AutoMigrate(Device{})
	if err != nil {
		logger.Fatal(err)
	}

	storage.Model(&Device{}).Where("online = true").Update("online", false)
}

func WebsocketMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Upgrade") == "websocket" {
			handleWebsocket(c)
			c.Abort()
		} else {
			c.Next()
		}
	}
}

func handleWebsocket(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	domain := GetDomain(strings.TrimPrefix(c.Request.URL.Path, "/"))
	if domain == nil {
		return
	}
	ws := &Websocket{conn: conn}
	conn.SetPingHandler(func(buffer string) error { return handlePingMessage(ws, domain, buffer) })

	for {
		ws.UpdateReadDeadline()
		messageType, buffer, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if messageType != websocket.BinaryMessage {
			continue
		}

		switch uint8(buffer[0]) {
		case AUTH:
			err = handleAuthMessage(ws, domain, buffer)
		case FORWARD:
			err = handleForwardMessage(ws, domain, buffer)
		case DHCP:
			err = handleDHCPMessage(ws, domain, buffer)
		case PEER:
			err = handlePeerConnMessage(ws, domain, buffer)
		case VMAC:
			err = handleVMacMessage(ws, domain, buffer)
		case DISCOVERY:
			err = handleDiscoveryMessage(ws, domain, buffer)
		case GENERAL:
			err = handleGeneralMessage(ws, domain, buffer)
		}

		if err != nil {
			logger.Debug(err)
			break
		}
	}

	domain.mutex.Lock()
	defer domain.mutex.Unlock()

	if device, ok := domain.wsDeviceMap[ws]; ok {
		if domain.ipWsMap[device.ip] == ws {
			delete(domain.ipWsMap, device.ip)
		}

		if device.Online {
			device.Online = false
			device.ConnUpdatedAt = time.Now()
			storage.Save(device)
		}

		delete(domain.wsDeviceMap, ws)
	}
}

func handlePingMessage(ws *Websocket, domain *Domain, buffer string) error {
	ws.UpdateReadDeadline()

	err := func() error {
		domain.mutex.RLock()
		defer domain.mutex.RUnlock()

		device, ok := domain.wsDeviceMap[ws]
		if !ok {
			return fmt.Errorf("ping failed: the client is not logged in: %v", buffer)
		}

		info := strings.Split(buffer, "::")
		if len(info) < 3 || info[0] != "candy" {
			return fmt.Errorf("ping failed: invalid format: %v", buffer)
		}

		device.OS = info[1]
		device.Version = info[2]

		if len(info) > 3 {
			device.Hostname = info[3]
		}

		return nil
	}()

	if err != nil && !ws.clientException {
		ws.clientException = true
		logger.Debug("client exception: ", err)
	}

	ws.WritePong([]byte(buffer))
	return nil
}

func handleAuthMessage(ws *Websocket, domain *Domain, buffer []byte) error {
	r := bytes.NewReader(buffer)
	message := &AuthMessage{}
	if err := struc.Unpack(r, message); err != nil {
		return err
	}

	if err := checkAuthMessage(domain, message); err != nil {
		return err
	}

	domain.mutex.Lock()
	defer domain.mutex.Unlock()

	device, ok := domain.wsDeviceMap[ws]
	if !ok {
		return fmt.Errorf("auth failed: vmac not received")
	}

	if domain.netID != domain.mask&message.IP {
		return fmt.Errorf("auth failed: network does not match")
	}

	if !checkIpConflict(domain, uint32ToIpString(message.IP), device.VMac) {
		return fmt.Errorf("auth failed: ip conflict: %v", uint32ToIpString(message.IP))
	}

	for oldWs, oldDevice := range domain.wsDeviceMap {
		if oldWs == ws {
			continue
		}

		if oldDevice.VMac == device.VMac {
			device.RX = oldDevice.RX
			device.TX = oldDevice.TX
			oldDevice.Online = false
			oldWs.conn.Close()
		}
	}

	device.ip = message.IP
	domain.ipWsMap[message.IP] = ws

	storage.Find(device)
	device.IP = uint32ToIpString(message.IP)
	device.Online = true
	device.ConnUpdatedAt = time.Now()
	device.Username = domain.Username
	storage.Save(device)
	return nil
}

func handleForwardMessage(ws *Websocket, domain *Domain, buffer []byte) error {
	domain.mutex.RLock()
	defer domain.mutex.RUnlock()

	device, ok := domain.wsDeviceMap[ws]
	if !ok {
		return fmt.Errorf("forward failed: conn is not logged in")
	}

	if !device.Online {
		return nil
	}

	r := bytes.NewReader(buffer)
	message := &ForwardMessage{}
	if err := struc.Unpack(r, message); err != nil {
		return err
	}

	if device.ip != message.Src {
		return fmt.Errorf("forward failed: source address does not match login information")
	}

	device.TX += uint64(len(buffer))

	if dstWs, ok := domain.ipWsMap[message.Dst]; ok {
		dstWs.WriteMessage(buffer)
		domain.wsDeviceMap[dstWs].RX += uint64(len(buffer))
	}

	broadcast := func() bool {
		if !domain.Broadcast {
			return false
		}
		if domain.netID|^domain.mask == message.Dst {
			return true
		}
		if message.Dst&0xF0000000 == 0xE0000000 {
			return true
		}
		return false
	}()

	if broadcast {
		for dstWs, dstDev := range domain.wsDeviceMap {
			if dstWs != ws && dstDev.Online {
				dstWs.WriteMessage(buffer)
				dstDev.RX += uint64(len(buffer))
			}
		}
	}

	return nil
}

func handleDHCPMessage(ws *Websocket, domain *Domain, buffer []byte) error {
	r := bytes.NewReader(buffer)
	message := &DHCPMessage{}
	if err := struc.Unpack(r, message); err != nil {
		return err
	}

	if err := checkDHCPMessage(domain, message); err != nil {
		return err
	}

	if domain.DHCP == "" {
		return fmt.Errorf("dhcp failed: DHCP is not enabled")
	}

	cidr := func(input []byte) string {
		return string(input[:bytes.IndexByte(input[:], 0)])
	}(message.Cidr)

	domain.mutex.RLock()
	defer domain.mutex.RUnlock()

	device, ok := domain.wsDeviceMap[ws]
	if !ok {
		return fmt.Errorf("dhcp failed: vmac not received")
	}
	ip, ipNet, err := net.ParseCIDR(cidr)
	needGenNewAddr := func() bool {
		if err != nil {
			return true
		}
		if binary.BigEndian.Uint32(ipNet.IP) != domain.netID {
			return true
		}
		if binary.BigEndian.Uint32(ipNet.Mask) != domain.mask {
			return true
		}
		devices := []Device{}
		storage.Where(&Device{Domain: domain.Name, IP: ip.String()}).Find(&devices)
		if len(devices) > 1 {
			return true
		}
		if len(devices) == 0 {
			return false
		}
		if len(devices) == 1 && devices[0].VMac == device.VMac {
			return false
		}
		return true
	}()

	var oldHostID = domain.hostID
	for needGenNewAddr {
		updateHostID(domain)
		result := storage.Where(&Device{Domain: domain.Name, IP: uint32ToIpString(domain.netID | domain.hostID)})
		if result.RowsAffected == 0 {
			break
		}
		if oldHostID == domain.hostID {
			return fmt.Errorf("dhcp failed: not enough addresses")
		}
	}

	if needGenNewAddr {
		ipNet := net.IPNet{
			IP:   make(net.IP, 4),
			Mask: make(net.IPMask, 4),
		}
		binary.BigEndian.PutUint32(ipNet.IP, domain.netID|domain.hostID)
		binary.BigEndian.PutUint32(ipNet.Mask, domain.mask)
		message.Cidr = []byte(ipNet.String())
	}

	var output bytes.Buffer
	struc.Pack(&output, message)
	ws.WriteMessage(output.Bytes())
	return nil
}

func handlePeerConnMessage(ws *Websocket, domain *Domain, buffer []byte) error {
	domain.mutex.RLock()
	defer domain.mutex.RUnlock()

	device, ok := domain.wsDeviceMap[ws]
	if !ok {
		return fmt.Errorf("peer conn failed: conn is not logged in")
	}

	r := bytes.NewReader(buffer)
	message := &PeerConnMessage{}
	if err := struc.Unpack(r, message); err != nil {
		return err
	}

	if device.ip != message.Src {
		return fmt.Errorf("peer conn failed: source address does not match login information")
	}

	UpdateLocation(device, uint32ToIpString(message.IP))

	if dst, ok := domain.ipWsMap[message.Dst]; ok {
		dst.WriteMessage(buffer)
	}

	return nil
}

func handleVMacMessage(ws *Websocket, domain *Domain, buffer []byte) error {
	r := bytes.NewReader(buffer)
	message := &VMacMessage{}
	if err := struc.Unpack(r, message); err != nil {
		return err
	}

	if err := checkVMacMessage(domain, message); err != nil {
		return err
	}

	domain.mutex.Lock()
	defer domain.mutex.Unlock()

	domain.wsDeviceMap[ws] = &Device{Domain: domain.Name, VMac: message.VMac}
	return nil
}

func handleDiscoveryMessage(ws *Websocket, domain *Domain, buffer []byte) error {
	domain.mutex.RLock()
	defer domain.mutex.RUnlock()

	device, ok := domain.wsDeviceMap[ws]
	if !ok || !device.Online {
		return nil
	}

	r := bytes.NewReader(buffer)
	message := &DiscoveryMessage{}
	if err := struc.Unpack(r, message); err != nil {
		return err
	}

	if device.ip != message.Src {
		return fmt.Errorf("discovery failed: source address does not match login information")
	}

	device.TX += uint64(len(buffer))

	if dstWs, ok := domain.ipWsMap[message.Dst]; ok {
		dstWs.WriteMessage(buffer)
		if dstDev, ok := domain.wsDeviceMap[dstWs]; ok {
			dstDev.RX += uint64(len(buffer))
		}
	}

	if uint32(0xFFFFFFFF) == message.Dst {
		for dstWs, dstDev := range domain.wsDeviceMap {
			if dstWs != ws && dstDev.Online {
				dstWs.WriteMessage(buffer)
				dstDev.RX += uint64(len(buffer))
			}
		}
	}

	return nil
}

func handleGeneralMessage(ws *Websocket, domain *Domain, buffer []byte) error {
	domain.mutex.RLock()
	defer domain.mutex.RUnlock()

	device, ok := domain.wsDeviceMap[ws]
	if !ok || !device.Online {
		return nil
	}

	r := bytes.NewReader(buffer)
	message := &GeneralMessage{}
	if err := struc.Unpack(r, message); err != nil {
		return err
	}

	if device.ip != message.Src {
		return fmt.Errorf("general failed: source address does not match login information")
	}

	device.TX += uint64(len(buffer))

	if dstWs, ok := domain.ipWsMap[message.Dst]; ok {
		dstWs.WriteMessage(buffer)
		if dstDev, ok := domain.wsDeviceMap[dstWs]; ok {
			dstDev.RX += uint64(len(buffer))
		}
	}

	if domain.Broadcast && uint32(0xFFFFFFFF) == message.Dst {
		for dstWs, dstDev := range domain.wsDeviceMap {
			if dstWs != ws && dstDev.Online {
				dstWs.WriteMessage(buffer)
				dstDev.RX += uint64(len(buffer))
			}
		}
	}

	return nil
}

func uint32ToIpString(ip uint32) string {
	var buffer []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(buffer, ip)
	return net.IP(buffer).String()
}

func checkIpConflict(domain *Domain, ip, vmac string) bool {
	device := &Device{Domain: domain.Name, IP: ip}
	result := storage.Where(device).Take(device)
	if result.Error == gorm.ErrRecordNotFound {
		return true
	}
	if result.Error == nil && device.VMac == vmac {
		return true
	}

	return false
}
