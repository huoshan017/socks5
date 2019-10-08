package socks5

import (
	"context"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"
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
		fmt.Fprintln(os.Stdout, err.Error())
		return err
	}

	var listener *net.TCPListener
	listener, err = net.ListenTCP("tcp", tcp_addr)
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		return err
	}

	t.listener = listener

	var c *net.TCPConn
	for {
		c, err = t.listener.AcceptTCP()
		if err != nil {
			fmt.Fprintln(os.Stdout, err.Error())
			return err
		}

		go t.serve(c)
		fmt.Fprintln(os.Stdout, "new connection from client: ", c.RemoteAddr(), "->", c.LocalAddr())
	}

	return nil
}

func (t *TcpServer) serve(conn *net.TCPConn) {
	var auth_req AuthRequest
	err := auth_req.Read(conn)
	if err != nil {
		conn.Close()
		fmt.Fprintln(os.Stdout, err.Error())
		return
	}

	auth_reply := NewAuthReply(0x00)
	err = auth_reply.Write(conn)
	if err != nil {
		conn.Close()
		fmt.Fprintln(os.Stdout, err.Error())
		return
	}

	var conn_cmd ConnectCmd
	err = conn_cmd.Read(conn)
	if err != nil {
		conn.Close()
		fmt.Fprintln(os.Stdout, err.Error())
		return
	}

	remote_addr := fmt.Sprintf("%v:%v", conn_cmd.Addr, conn_cmd.Port)
	var remote_conn net.Conn
	var reply_res uint8
	if conn_cmd.Cmd != CMD_CONNECT {
		reply_res = REPLY_COMMAND_NOT_SUPPORTED
	} else {
		for {
			remote_conn, err = net.Dial("tcp", remote_addr)
			if err != nil {
				if net_err, ok := err.(net.Error); ok {
					if net_err.Temporary() {
						time.Sleep(time.Second)
						continue
					}
					reply_res = _get_reply_error_code(net_err)
				} else {
					reply_res = REPLY_SOCKS_SERVER_FAILURE
				}
				fmt.Fprintln(os.Stdout, err.Error())
			} else {
				reply_res = REPLY_SUCCEED
			}
			break
		}
	}

	conn_reply := NewConnectReply(reply_res, conn_cmd.AddrType, conn_cmd.Addr, 0)
	err = conn_reply.Write(conn)
	if err != nil {
		conn.Close()
		fmt.Fprintln(os.Stdout, "%v", err.Error())
		return
	}

	if reply_res != REPLY_SUCCEED {
		conn.Close()
		fmt.Fprintln(os.Stdout, "connect remote host reply failed", reply_res, ", close connection to sock client")
		return
	}

	fmt.Fprintln(os.Stdout, "connect remote host", remote_addr, "success")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// read from socks client and write to remote server
	go func(ctx context.Context) {
		var local_buf [1024]byte
		//conn.SetReadDeadline(time.Now().Add(time.Millisecond * 1000))
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			read_bytes, e := conn.Read(local_buf[:])
			if e != nil {
				fmt.Fprintln(os.Stdout, "read from socks client err: ", e.Error())
				break
			}
			_, e = remote_conn.Write(local_buf[:read_bytes])
			if e != nil {
				fmt.Fprintln(os.Stdout, "write to remote server err: ", e.Error())
				break
			}
		}
	}(ctx)

	// read from remote server and write to socks client
	var remote_buf [4096]byte
	//remote_conn.SetReadDeadline(time.Now().Add(time.Millisecond * 2000))
	for {
		read_bytes, e := remote_conn.Read(remote_buf[:])
		if e != nil {
			fmt.Fprintln(os.Stdout, "read from remote server err: ", e.Error())
			break
		}
		_, e = conn.Write(remote_buf[:read_bytes])
		if e != nil {
			fmt.Fprintln(os.Stdout, "write to socks client err: ", e.Error())
			break
		}
	}
}

func _get_reply_error_code(net_err net.Error) uint8 {
	if net_err.Timeout() {
		return REPLY_TTL_EXPIRED
	}

	var reply uint8 = REPLY_SOCKS_SERVER_FAILURE
	op_err, ok := net_err.(*net.OpError)
	if !ok {
		return reply
	}

	switch t := op_err.Err.(type) {
	case *net.AddrError:
		reply = REPLY_ADDRESS_TYPE_NOT_SUPPORTED
	case *os.SyscallError:
		errno, o := t.Err.(syscall.Errno)
		if o {
			fmt.Fprintln(os.Stdout, "syscall errno: ", errno)
			switch errno {
			case syscall.ECONNREFUSED:
				reply = REPLY_CONNECTION_REFUSED
			case syscall.ENETUNREACH:
				reply = REPLY_NETWORK_UNREACHABLE
			case syscall.EHOSTUNREACH:
				reply = REPLY_HOST_UNREACHABLE
			case syscall.ENOTCONN:
				reply = REPLY_CONNECTION_NOT_ALLOW
			case syscall.ETIMEDOUT:
				reply = REPLY_TTL_EXPIRED
			}
		}
	}
	return reply
}
