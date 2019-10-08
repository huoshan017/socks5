package socks5

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

func get_reply_error_code(net_err net.Error) uint8 {
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
