package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/yinqiwen/ssf"
)

func main() {
	ss := flag.String("cluster", "127.0.0.1:48100,127.0.0.1:48101,127.0.0.1:48102", "cluster servers")
	flag.Parse()

	servers := strings.Split(*ss, ",")
	var conns []net.Conn
	for _, s := range servers {
		c, err := net.Dial("tcp", s)
		if nil != err {
			fmt.Printf("Failed to connect %s for reason:%v\n", err)
		} else {
			conns = append(conns, c)
		}
	}
	if len(conns) == 0 {
		fmt.Printf("No connected server to send text event.\n")
		return
	}
	cursor := 0
	bc := bufio.NewReader(os.Stdin)
	scanner := bufio.NewScanner(bc)
	for scanner.Scan() {
		line := scanner.Text()
		rawMsg := ssf.NewRawMessage(line)
		if cursor >= len(conns) {
			cursor = 0
		}
		c := conns[cursor]
		cursor++
		err := ssf.Notify(rawMsg, 1, c)
		if nil != err {
			fmt.Printf("Failed to send text event to %v for reason:%v.\n", c.RemoteAddr(), err)
		}
	}

	for _, c := range conns {
		c.Close()
	}
}
