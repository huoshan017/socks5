package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/huoshan017/socks5"
)

func main() {
	proxy_addr := flag.String("proxy_addr", "127.0.0.1:9000", "")
	flag.Parse()

	client, err := socks5.NewTcpClient(*proxy_addr)
	if err != nil {
		fmt.Fprintf(os.Stdout, "create tcp client err: %v", err.Error())
		return
	}
	err = client.Auth(0x00)
	if err != nil {
		fmt.Fprintf(os.Stdout, "auth err: %v", err.Error())
		return
	}
	err = client.Connect("www.baidu.com", 80)
	if err != nil {
		fmt.Fprintf(os.Stdout, "connect err: %v", err.Error())
		return
	}
}
