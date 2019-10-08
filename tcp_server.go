package socks5

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
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

		go serve(c)
		fmt.Fprintln(os.Stdout, "new connection from client: ", c.RemoteAddr(), "->", c.LocalAddr())
	}

	return nil
}

func serve(conn *net.TCPConn) {
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
					reply_res = get_reply_error_code(net_err)
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
	go read_and_write(ctx, conn, remote_conn, 5000, 0, 1024)

	// read from remote server and write to socks client
	read_and_write(ctx, remote_conn, conn, 5000, 0, 4096)

	conn.Close()
	remote_conn.Close()

	// read from socks client and write to remote server
	/*go func(ctx context.Context) {
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
				if e != io.EOF {
					fmt.Fprintln(os.Stdout, "read from socks client err: ", e.Error())
				}
				break
			}
			_, e = remote_conn.Write(local_buf[:read_bytes])
			if e != nil {
				if e != io.EOF {
					fmt.Fprintln(os.Stdout, "write to remote server err: ", e.Error())
				}
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
			if e != io.EOF {
				fmt.Fprintln(os.Stdout, "read from remote server err: ", e.Error())
			}
			break
		}
		_, e = conn.Write(remote_buf[:read_bytes])
		if e != nil {
			if e != io.EOF {
				fmt.Fprintln(os.Stdout, "write to socks client err: ", e.Error())
			}
			break
		}
	}*/
}

func read_and_write(ctx context.Context, read_conn, write_conn net.Conn, read_deadline, write_deadline int, buf_len int) {
	buf := make([]byte, buf_len)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if read_deadline > 0 {
			read_conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(read_deadline)))
		}
		if write_deadline > 0 {
			write_conn.SetWriteDeadline(time.Now().Add(time.Millisecond * time.Duration(write_deadline)))
		}
		read_bytes, e := read_conn.Read(buf[:])
		if e != nil {
			if e != io.EOF {
				fmt.Fprintln(os.Stdout, "read from socks client err: ", e.Error())
			}
			break
		}
		_, e = write_conn.Write(buf[:read_bytes])
		if e != nil {
			if e != io.EOF {
				fmt.Fprintln(os.Stdout, "write to remote server err: ", e.Error())
			}
			break
		}
	}
}
