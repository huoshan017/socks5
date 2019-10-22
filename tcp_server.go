package socks5

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"time"
)

const (
	DefaultSocksReadDeadline     = 2000 // milliseconds
	DefaultSocksWriteDeadline    = 0    // milliseconds
	DefaultRemoteReadDeadline    = 5000 // milliseconds
	DefaultRemoteWriteDeadline   = 0    // milliseconds
	DefaultSocks2RemoteBufLen    = 1024
	DefaultRemote2SocksBufLen    = 4096
	DefaultReadWriteLoopInterval = 1 // milliseconds
)

type ServerConfig struct {
	ListenAddr            string
	SocksReadDeadline     int
	SocksWriteDeadline    int
	RemoteReadDeadline    int
	RemoteWriteDeadline   int
	Socks2RemoteBuflen    int
	Remote2SocksBuflen    int
	ReadWriteLoopInterval int
}

type TcpServer struct {
	listener *net.TCPListener
	config   *ServerConfig
}

func NewTcpServer(config *ServerConfig) *TcpServer {
	if config.SocksReadDeadline == 0 {
		config.SocksReadDeadline = DefaultSocksReadDeadline
	}
	if config.SocksWriteDeadline == 0 {
		config.SocksWriteDeadline = DefaultSocksWriteDeadline
	}
	if config.RemoteReadDeadline == 0 {
		config.RemoteReadDeadline = DefaultRemoteReadDeadline
	}
	if config.RemoteWriteDeadline == 0 {
		config.RemoteWriteDeadline = DefaultRemoteWriteDeadline
	}
	if config.Socks2RemoteBuflen == 0 {
		config.Socks2RemoteBuflen = DefaultSocks2RemoteBufLen
	}
	if config.Remote2SocksBuflen == 0 {
		config.Remote2SocksBuflen = DefaultRemote2SocksBufLen
	}
	if config.ReadWriteLoopInterval == 0 {
		config.ReadWriteLoopInterval = DefaultReadWriteLoopInterval
	}
	return &TcpServer{
		config: config,
	}
}

func (t *TcpServer) Start() error {
	tcp_addr, err := net.ResolveTCPAddr("tcp", t.config.ListenAddr)
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

	var l = 2
	c := make(chan struct{}, l)

	// read from socks client and write to remote server
	go read_write_loop(ctx, conn, remote_conn, t.config.SocksReadDeadline, t.config.RemoteWriteDeadline, t.config.Socks2RemoteBuflen, c, t.config.ReadWriteLoopInterval)

	// read from remote server and write to socks client
	go read_write_loop(ctx, remote_conn, conn, t.config.RemoteReadDeadline, t.config.SocksWriteDeadline, t.config.Remote2SocksBuflen, c, t.config.ReadWriteLoopInterval)

	for i := 0; i < l; i++ {
		<-c
	}

	close(c)

	conn.Close()
	remote_conn.Close()
}

func read_write_loop(ctx context.Context, read_conn, write_conn net.Conn, read_deadline, write_deadline int, buf_len int, c chan struct{}, read_write_interval int) {
	defer func() {
		c <- struct{}{}
	}()

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
		/*if write_deadline > 0 {
			write_conn.SetWriteDeadline(time.Now().Add(time.Millisecond * time.Duration(write_deadline)))
		}*/
		read_bytes, e := read_conn.Read(buf[:])
		if e != nil {
			if e != io.EOF {
				if read_deadline > 0 {
					ne, o := e.(net.Error)
					if o && ne.Timeout() {
						fmt.Fprintln(os.Stdout, "!!!!!!!!!!!!!!!!!!!!!!!!!! goroutines: ", runtime.NumGoroutine())
						goto ReadWritePause
					}
				}
				fmt.Fprintln(os.Stdout, "read from socks client err: ", e.Error())
			}
			break
		}
		if read_bytes > 0 {
			_, e = write_conn.Write(buf[:read_bytes])
			if e != nil {
				/*if write_deadline > 0 {
					ne := e.(net.Error)
					if ne != nil && ne.Timeout() {
						goto ReadWritePause
					}
				}*/
				if e != io.EOF {
					fmt.Fprintln(os.Stdout, "write to remote server err: ", e.Error())
				}
				break
			}
		}
	ReadWritePause:
		time.Sleep(time.Millisecond * time.Duration(read_write_interval))
	}
}
