package socks5

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
)

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
	CMD_UDP_ASSOCIATE = 0x03

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
)

type AuthRequest struct {
	Ver      uint8
	NMethods uint8
	Methods  []uint8
}

func NewAuthRequest(methods []uint8) *AuthRequest {
	return &AuthRequest{Ver: VERSION, NMethods: uint8(len(methods)), Methods: methods}
}

func (a *AuthRequest) Write(conn net.Conn) error {
	buf := []byte{a.Ver, uint8(len(a.Methods))}
	buf = append(buf, a.Methods...)
	_, err := conn.Write(buf)
	return err
}

func (a *AuthRequest) Read(conn net.Conn) error {
	var buf [2]byte
	_, err := io.ReadFull(conn, buf[:])
	if err != nil {
		return err
	}
	l := int(buf[1])
	methods := make([]byte, l)
	_, err = io.ReadFull(conn, methods)
	if err != nil {
		return err
	}
	a.Ver = buf[0]
	a.NMethods = uint8(l)
	a.Methods = methods
	return nil
}

type AuthReply struct {
	Ver    uint8
	Method uint8
}

func NewAuthReply(method int) *AuthReply {
	return &AuthReply{Ver: VERSION, Method: uint8(method)}
}

func (a *AuthReply) Write(conn net.Conn) error {
	_, err := conn.Write([]byte{a.Ver, a.Method})
	return err
}

func (a *AuthReply) Read(conn net.Conn) error {
	var buf [2]byte
	_, err := io.ReadFull(conn, buf[:])
	if err != nil {
		return err
	}
	a.Ver = buf[0]
	a.Method = buf[1]
	return nil
}

type ConnectCmd struct {
	Ver      uint8
	Cmd      uint8
	AddrType uint8
	Addr     string
	Port     uint16
}

func _parse_host(host string) (addr_type int, addr string) {
	ip := net.ParseIP(host)
	if ip != nil {
		ipv4 := ip.To4()
		if ipv4 != nil {
			addr_type = CMD_ADDR_IPV4
			addr = ipv4.String()
		} else {
			addr_type = CMD_ADDR_IPV6
			addr = ip.To16().String()
		}
	} else {
		addr_type = CMD_ADDR_DOMAIN
		addr = host
	}
	return
}

func NewConncectCmd(cmd uint8, remote_host string, remote_port uint16) *ConnectCmd {
	addr_type, addr := _parse_host(remote_host)
	return &ConnectCmd{
		Ver:      VERSION,
		Cmd:      uint8(cmd),
		AddrType: uint8(addr_type),
		Addr:     addr,
		Port:     remote_port,
	}
}

func (c *ConnectCmd) Write(conn net.Conn) error {
	var buf = []byte{c.Ver, c.Cmd, 0x00, c.AddrType}
	if c.AddrType == CMD_ADDR_IPV4 || c.AddrType == CMD_ADDR_IPV6 {
		buf = append(buf, []byte(c.Addr)...)
	} else {
		buf = append(buf, byte(len(c.Addr)))
	}
	buf = append(buf, []byte{byte((c.Port >> 8) & 0xff), byte(c.Port & 0xff)}...)
	_, err := conn.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func (c *ConnectCmd) Read(conn net.Conn) error {
	var buf [4]byte
	_, err := io.ReadFull(conn, buf[:])
	if err != nil {
		return err
	}
	c.Ver = buf[0]
	c.Cmd = buf[1]
	c.AddrType = buf[3]
	if c.AddrType == CMD_ADDR_IPV4 {
		var ip_port [net.IPv4len + 2]byte
		_, err = io.ReadFull(conn, ip_port[:])
		if err != nil {
			return err
		}
		c.Addr = net.IP(ip_port[:net.IPv4len]).String()
		c.Port = binary.BigEndian.Uint16(ip_port[net.IPv4len : net.IPv4len+2]) //uint16((ip_port[net.IPv4len]<<8)&0xff00) | uint16(ip_port[net.IPv4len+1]&0xff)
	} else if c.AddrType == CMD_ADDR_IPV6 {
		var ip_port [net.IPv6len + 2]byte
		_, err = io.ReadFull(conn, ip_port[:])
		if err != nil {
			return err
		}
		c.Addr = net.IP(ip_port[:net.IPv6len]).String()
		c.Port = binary.BigEndian.Uint16(ip_port[net.IPv6len : net.IPv6len+2]) //uint16((ip_port[net.IPv6len]<<8)&0xff00) | uint16(ip_port[net.IPv6len+1]&0xff)
	} else {
		_, err = io.ReadFull(conn, buf[:1])
		if err != nil {
			return err
		}
		ip_len := buf[0]
		ip_port := make([]byte, ip_len+2)
		_, err = io.ReadFull(conn, ip_port)
		if err != nil {
			return err
		}
		c.Addr = string(ip_port[:ip_len])
		c.Port = binary.BigEndian.Uint16(ip_port[ip_len : ip_len+2]) //uint16((ip_port[ip_len]<<8)&0xff00) | uint16(ip_port[ip_len+1]&0xff)
	}
	return nil
}

type ConnectReply struct {
	Ver      uint8
	Reply    uint8
	AddrType uint8
	BindAddr string
	BindPort uint16
}

func NewConnectReply(reply uint8, addr_type uint8, bind_addr string, bind_port uint16) *ConnectReply {
	return &ConnectReply{Ver: VERSION, Reply: reply, AddrType: addr_type, BindAddr: bind_addr, BindPort: bind_port}
}

func (c *ConnectReply) Write(conn net.Conn) error {
	var buf = []byte{c.Ver, c.Reply, 0x00, c.AddrType}
	if c.BindAddr != "" {
		addr_type, addr := _parse_host(c.BindAddr)
		if addr_type == CMD_ADDR_IPV4 || addr_type == CMD_ADDR_IPV6 {
			buf = append(buf, addr...)
		} else {
			buf = append(buf, byte(len(addr)))
			buf = append(buf, addr...)
		}
		buf = append(buf, []byte{byte((c.BindPort >> 8) & 0xff), byte(c.BindPort & 0xff)}...)
	} else {
		buf = append(buf, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
	}

	fmt.Fprintln(os.Stdout, "ConnectReply Write buf:", buf)

	_, err := conn.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func (c *ConnectReply) Read(conn net.Conn) error {
	var buf [4]byte
	_, err := io.ReadFull(conn, buf[:])
	if err != nil {
		return err
	}
	c.Ver = buf[0]
	c.Reply = buf[1]
	c.AddrType = buf[3]
	return nil
}
