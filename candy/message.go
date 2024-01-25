package candy

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"
)

const (
	AUTH      uint8 = 0
	FORWARD   uint8 = 1
	DHCP      uint8 = 2
	PEER      uint8 = 3
	VMAC      uint8 = 4
	DISCOVERY uint8 = 5
	GENERAL   uint8 = 255
)

type AuthMessage struct {
	Type      uint8    `struc:"uint8"`
	IP        uint32   `struc:"uint32"`
	Timestamp int64    `struc:"int64"`
	Hash      [32]byte `struc:"[32]byte"`
}

type ForwardMessage struct {
	Type   uint8    `struc:"uint8"`
	Unused [12]byte `struc:"[12]byte"`
	Src    uint32   `struc:"uint32"`
	Dst    uint32   `struc:"uint32"`
}

type DHCPMessage struct {
	Type      uint8    `struc:"uint8"`
	Timestamp int64    `struc:"int64"`
	Cidr      []byte   `struc:"[32]byte"`
	Hash      [32]byte `struc:"[32]byte"`
}

type PeerConnMessage struct {
	Type uint8  `struc:"uint8"`
	Src  uint32 `struc:"uint32"`
	Dst  uint32 `struc:"uint32"`
	IP   uint32 `struc:"uint32"`
	Port uint16 `struc:"uint16"`
}

type VMacMessage struct {
	Type      uint8    `struc:"uint8"`
	VMac      string   `struc:"[16]byte"`
	Timestamp int64    `struc:"int64"`
	Hash      [32]byte `struc:"[32]byte"`
}

type DiscoveryMessage struct {
	Type uint8  `struc:"uint8"`
	Src  uint32 `struc:"uint32"`
	Dst  uint32 `struc:"uint32"`
}

type GeneralMessage struct {
	Type    uint8  `struc:"uint8"`
	Subtype uint8  `struc:"uint8"`
	Extra   uint16 `struc:"uint16"`
	Src     uint32 `struc:"uint32"`
	Dst     uint32 `struc:"uint32"`
}

func absInt64(a, b int64) int64 {
	if a > b {
		return a - b
	}
	return b - a
}

func checkAuthMessage(domain *Domain, message *AuthMessage) error {
	if absInt64(time.Now().Unix(), message.Timestamp) > 30 {
		return fmt.Errorf("auth check failed: timestamp: %v", message.Timestamp)
	}

	reported := message.Hash

	var data []byte
	data = append(data, domain.Password...)
	data = binary.BigEndian.AppendUint32(data, message.IP)
	data = binary.BigEndian.AppendUint64(data, uint64(message.Timestamp))

	if sha256.Sum256([]byte(data)) != reported {
		return fmt.Errorf("auth check failed: hash does not match")
	}
	return nil
}

func checkDHCPMessage(domain *Domain, message *DHCPMessage) error {
	if absInt64(time.Now().Unix(), message.Timestamp) > 30 {
		return fmt.Errorf("dhcp check failed: timestamp: %v", message.Timestamp)
	}

	reported := message.Hash

	var data []byte
	data = append(data, domain.Password...)
	data = binary.BigEndian.AppendUint64(data, uint64(message.Timestamp))

	if sha256.Sum256([]byte(data)) != reported {
		return fmt.Errorf("dhcp check failed: hash does not match")
	}
	return nil
}

func checkVMacMessage(domain *Domain, message *VMacMessage) error {
	if absInt64(time.Now().Unix(), message.Timestamp) > 30 {
		return fmt.Errorf("vmac check failed: timestamp: %v", message.Timestamp)
	}

	if _, err := strconv.ParseUint(message.VMac, 16, 64); err != nil {
		return fmt.Errorf("vmac check failed: invalid vmac")
	}

	reported := message.Hash

	var data []byte
	data = append(data, domain.Password...)
	data = append(data, message.VMac...)
	data = binary.BigEndian.AppendUint64(data, uint64(message.Timestamp))

	if sha256.Sum256([]byte(data)) != reported {
		return fmt.Errorf("vmac check failed: hash does not match")
	}
	return nil
}
