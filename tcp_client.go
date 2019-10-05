package socks5

import (
	"net"
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
	request := NewAuthRequest([]uint8{uint8(method)})
	err := request.Write(c.conn)
	if err != nil {
		return err
	}

	var reply AuthReply
	err = reply.Read(c.conn)
	if err != nil {
		return err
	}

	if reply.Ver != VERSION {
		return ErrSocksVersionNotSupport
	}

	if reply.Method == METHOD_NO_ACCEPTABLE {
		return ErrNotSupportClientMethod
	}

	return nil
}

func (c *TcpClient) Connect(remote_host string, remote_port uint16) error {
	conn_cmd := NewConncectCmd(CMD_CONNECT, remote_host, remote_port)
	err := conn_cmd.Write(c.conn)
	if err != nil {
		return err
	}

	var conn_reply ConnectReply
	err = conn_reply.Read(c.conn)
	if err != nil {
		return err
	}

	if conn_reply.Ver != VERSION {
		return ErrSocksVersionNotSupport
	}

	switch conn_reply.Reply {
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

	return nil
}
