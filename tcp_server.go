package socks5

import (
	"net"
)

const (
	DefaultReadLen = 4096
)

type TcpServer struct {
	listener *net.TCPListener
}

func NewTcpServer() *TcpServer {
	return &TcpServer{}
}

func (t *TcpServer) Start(listen_addr string) error {
	tcp_addr, err := net.ResolveTCPAddr("tcp", listen_addr)
	if err != nil {
		return err
	}

	var listener *net.TCPListener
	listener, err = net.ListenTCP("tcp", tcp_addr)
	if err != nil {
		return err
	}

	t.listener = listener

	var c *net.TCPConn
	for {
		c, err = t.listener.AcceptTCP()
		if err != nil {
			return err
		}

		go t.serve(c)
	}

	return nil
}

func (t *TcpServer) serve(conn *net.TCPConn) {

}
