package socks5

const (
	VERSION = 5

	METHOD_NO_AUTH          = 0x00
	METHOD_AUTH_GSSAPI      = 0x01
	METHOD_USER_PASS        = 0x02
	METHOD_IANA             = 0x03
	METHOD_RESERVED_PRIVATE = 0x80
	METHOD_NO_ACCEPTABLE    = 0xFF

	CMD_CONNECT       = 0x01
	CMD_BIND          = 0x02
	CMD_DOMAIN        = 0x03
	CMD_UDP_ASSOCIATE = 0x04

	CMD_RESERVED = 0x00

	CMD_ADDR_IPV4   = 0x01
	CMD_ADDR_DOMAIN = 0x03
	CMD_ADDR_IPV6   = 0x04

	REPLY_SUCCEED                    = 0x00
	REPLY_SOCKS_SERVER_FAILURE       = 0x01
	REPLY_CONNECTION_NOT_ALLOW       = 0x02
	REPLY_NETWORK_UNREACHABLE        = 0x03
	REPLY_HOST_UNREACHABLE           = 0x04
	REPLY_CONNECTION_REFUSED         = 0x05
	REPLY_TTL_EXPIRED                = 0x06
	REPLY_COMMAND_NOT_SUPPORTED      = 0x07
	REPLY_ADDRESS_TYPE_NOT_SUPPORTED = 0x08
	REPLY_UNASSIGNED                 = 0x09
)

type AuthRequest struct {
	Ver      uint8
	NMethods uint8
	Methods  []uint8
}

type AuthReply struct {
	Ver    uint8
	Method uint8
}

type ConnectCmd struct {
	Ver      uint8
	Cmd      uint8
	AddrType uint8
	Addr     string
	Port     uint16
}

type ConnectReply struct {
	Ver      uint8
	Reply    uint8
	AddrType uint8
	Addr     string
	Port     uint16
}
