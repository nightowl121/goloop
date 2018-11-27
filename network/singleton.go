package network

import (
	"errors"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

var (
	ErrAlreadyListened = errors.New("Already listened")
	ErrAlreadyClosed   = errors.New("Already closed")
)
var (
	singletonTransport module.NetworkTransport
	singletonManagers  = make(map[string]module.NetworkManager)
	singletonConfig    *Config
)

var (
	singletonLoggerExcludes = &[]string{"onPacket"}
)

const (
	DefaultTransportNet       = "tcp4"
	DefaultMembershipName     = ""
	DefaultPacketBufferSize   = 4096 //bufio.defaultBufSize=4096
	DefaultDiscoveryPeriodSec = 1
	DefaultSeedPeriodSec      = 1
)

// const (
// 	PROTO_CONTOL     = 0x0000
// 	PROTO_DEF_MEMBER = 0x0100
// )

// const (
// 	PROTO_AUTH_HS1 = 0x0101
// 	PROTO_AUTH_HS2 = 0x0201
// 	PROTO_AUTH_HS3 = 0x0301
// 	PROTO_AUTH_HS4 = 0x0401
// )

// const (
// 	PROTO_CHAN_JOIN_REQ  = 0x0501
// 	PROTO_CHAN_JOIN_RESP = 0x0601
// )

// const (
// 	PROTO_P2P_QUERY        = 0x0701
// 	PROTO_P2P_QUERY_RESULT = 0x0801
// )

var (
	PROTO_CONTOL           module.ProtocolInfo = protocolInfo(0x0000)
	PROTO_DEF_MEMBER       module.ProtocolInfo = protocolInfo(0x0100)
	PROTO_AUTH_KEY_REQ     module.ProtocolInfo = protocolInfo(0x0100)
	PROTO_AUTH_KEY_RESP    module.ProtocolInfo = protocolInfo(0x0200)
	PROTO_AUTH_SIGN_REQ    module.ProtocolInfo = protocolInfo(0x0300)
	PROTO_AUTH_SIGN_RESP   module.ProtocolInfo = protocolInfo(0x0400)
	PROTO_CHAN_JOIN_REQ    module.ProtocolInfo = protocolInfo(0x0501)
	PROTO_CHAN_JOIN_RESP   module.ProtocolInfo = protocolInfo(0x0601)
	PROTO_P2P_QUERY        module.ProtocolInfo = protocolInfo(0x0701)
	PROTO_P2P_QUERY_RESULT module.ProtocolInfo = protocolInfo(0x0801)
	PROTO_P2P_CONN_REQ     module.ProtocolInfo = protocolInfo(0x0901)
	PROTO_P2P_CONN_RESP    module.ProtocolInfo = protocolInfo(0x0A01)
)

type Config struct {
	ListenAddress string
	SeedAddress   string
	RoleSeed      bool
	RoleRoot      bool
	PrivateKey    *crypto.PrivateKey
}

func GetConfig() *Config {
	if singletonConfig == nil {
		//TODO Read from file or DB
		priK, _ := crypto.GenerateKeyPair()
		singletonConfig = &Config{
			ListenAddress: "127.0.0.1:8080",
			PrivateKey:    priK,
		}

	}
	return singletonConfig
}

func GetTransport() module.NetworkTransport {
	if singletonTransport == nil {
		c := GetConfig()
		w, _ := common.WalletFromPrivateKey(c.PrivateKey)
		singletonTransport = NewTransport(c.ListenAddress, w)
	}
	return singletonTransport
}

func GetManager(channel string) module.NetworkManager {
	nm, ok := singletonManagers[channel]
	if !ok {
		c := GetConfig()
		t := GetTransport()
		m := NewManager(channel, t)

		r := PeerRoleFlag(p2pRoleNone)
		if c.RoleSeed {
			r.SetFlag(p2pRoleSeed)
		}
		if c.RoleRoot {
			r.SetFlag(p2pRoleRoot)
		}
		m.(*manager).p2p.setRole(r)
		if c.SeedAddress != "" {
			m.(*manager).p2p.seeds.Add(NetAddress(c.SeedAddress))
		}
		nm = m
		singletonManagers[channel] = m
	}
	return nm
}
