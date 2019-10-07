package main

import (
	"flag"
	"log"

	"github.com/huoshan017/socks5"
)

func main() {
	proxy_addr := flag.String("listen", "127.0.0.1:9000", "")
	flag.Parse()
	server := socks5.NewTcpServer()
	err := server.Start(*proxy_addr)
	if err != nil {
		log.Fatalf("server start err: %v", err.Error())
	}
}
