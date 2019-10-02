package socks5

import (
	"net"
	"strconv"
	"strings"
)

type TcpClient struct {
	conn       *net.TCPConn
	proxy_addr string
	bind_addr  string
	bind_port  uint16
}

func NewTcpClient(proxy_addr string) (*TcpClient, error) {
	tcp_addr, err := net.ResolveTCPAddr("tcp", proxy_addr)
	if err != nil {
		return nil, err
	}

	var conn *net.TCPConn
	conn, err = net.DialTCP("tcp", nil, tcp_addr)
	if err != nil {
		return nil, err
	}

	return &TcpClient{
		conn:       conn,
		proxy_addr: proxy_addr,
	}, nil
}

func (c *TcpClient) GetConn() *net.TCPConn {
	return c.conn
}

func (c *TcpClient) Auth(method int) error {
	err := write_all(c.conn, []byte{VERSION, 0x01, byte(method)})
	if err != nil {
		return err
	}

	var buf [2]byte
	err = read_all(c.conn, buf[:])
	if err != nil {
		return err
	}

	if buf[0] != VERSION {
		return ErrSocksVersionNotSupport
	}

	if buf[1] == METHOD_NO_ACCEPTABLE {
		return ErrNotSupportClientMethod
	}

	return nil
}

func (c *TcpClient) Connect(remote_addr string) error {
	s := strings.Split(remote_addr, ":")
	if len(s) < 2 {
		return ErrAddrFormatInvalid
	}

	ip := net.ParseIP(s[0])
	port, err := strconv.Atoi(s[1])
	if err != nil {
		return err
	}
	var addr_type int
	var addr []byte
	if ip != nil {
		if ip.To4() != nil {
			addr_type = CMD_ADDR_IPV4
			addr = []byte(ip.To4().String())
		} else {
			addr_type = CMD_ADDR_IPV6
			addr = []byte(ip.To16().String())
		}
	} else {
		addr_type = CMD_ADDR_DOMAIN
		addr = []byte{byte(len(s[0]))}
		addr = append(addr, []byte(s[0])...)
	}

	var data = []byte{VERSION, CMD_CONNECT, 0x01, byte(addr_type)}
	data = append(data, addr...)
	data = append(data, byte(port>>8&0xff))
	data = append(data, byte(port&0xff))
	err = write_all(c.conn, data)
	if err != nil {
		return err
	}

	var buf [4]byte
	err = read_all(c.conn, buf[:])
	if err != nil {
		return err
	}

	if buf[0] != VERSION {
		return ErrSocksVersionNotSupport
	}

	switch buf[1] {
	case REPLY_SOCKS_SERVER_FAILURE:
		return ErrSocksServerFailure
	case REPLY_CONNECTION_NOT_ALLOW:
		return ErrRemoteConnectionNotAllow
	case REPLY_NETWORK_UNREACHABLE:
		return ErrRemoteNetworkUnreachable
	case REPLY_HOST_UNREACHABLE:
		return ErrRemoteHostUnreachable
	case REPLY_CONNECTION_REFUSED:
		return ErrRemoteConnectionRefused
	case REPLY_TTL_EXPIRED:
		return ErrRemoteTTLExpired
	case REPLY_COMMAND_NOT_SUPPORTED:
		return ErrCommandNotSupported
	case REPLY_ADDRESS_TYPE_NOT_SUPPORTED:
		return ErrAddressTypeNotSupported
	}

	if buf[3] == CMD_ADDR_DOMAIN {
	} else if buf[3] == CMD_ADDR_IPV6 {
	}

	return nil
}
