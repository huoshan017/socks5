package socks5

import (
	"net"
)

func write_all(conn net.Conn, buf []byte) error {
	var n, w int
	var err error
	for {
		n, err = conn.Write(buf[w:])
		if err != nil {
			return err
		}
		w += n
		if w == len(buf) {
			break
		}
	}
	return nil
}

func read_all(conn net.Conn, buf []byte) error {
	var n, r int
	var err error
	for {
		n, err = conn.Read(buf[r:])
		if err != nil {
			return err
		}
		r += n
		if r == len(buf) {
			break
		}
	}
	return nil
}
