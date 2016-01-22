package main

import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

type Incoming struct {
	Port int64
	Conn *net.TCPConn
}

func IncomingFromConn(c *net.TCPConn) (i *Incoming) {
	addr := c.LocalAddr().String()
	splitted := strings.Split(addr, ":")
	port, _ := strconv.ParseInt(splitted[1], 10, 64)
	i = &Incoming{
		Port: port,
		Conn: c,
	}
	return i
}

var (
	destination_host = kingpin.Flag("destination-host", "Destination host").Short('d').Default("127.0.0.1").IP()
	bind_host        = kingpin.Flag("bind", "Bind (listen) host").Short('b').Default("0.0.0.0").IP()
	min_port         = kingpin.Flag("min-port", "minimum port value").Short('m').Default("1").Int()
	max_port         = kingpin.Flag("max-port", "maximum port value").Short('M').Default("65535").Int()
	connections      map[int64]*net.TCPConn
	incoming         chan *Incoming
)

func init() {
	connections = make(map[int64]*net.TCPConn)
	// more incoming connections than the number of available ports
	incoming = make(chan *Incoming, 100000)
}

func listener(host string, port int, c chan *Incoming) {
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Printf("cannot resolve %s", err)
		return
	}
	l, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Printf("cannot listen %s", err)
		return
	}
	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			fmt.Printf("error accepting %s", err)
			continue
		}
		c <- IncomingFromConn(conn)
	}
}

func broker(dst, src *net.TCPConn) {
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("copy error: %s", err)
	}
}

func proxy(in chan *Incoming) {
	for incoming := range in {
		out, ok := connections[incoming.Port]
		if !ok {
			out_addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", (*destination_host).String(), incoming.Port))
			if err != nil {
				log.Printf("cannot resolve destination %s", err)
				continue
			}
			out, err = net.DialTCP("tcp4", nil, out_addr)
			if err != nil {
				log.Printf("cannot dial %s", err)
				continue
			}
			connections[incoming.Port] = out
		}
		// start the copy on connection until close from source
		go broker(out, incoming.Conn)
		go broker(incoming.Conn, out)
	}
}

func main() {
	kingpin.Parse()
	log.Printf("sending incoming queries to %s", *destination_host)

	for p := *min_port; p <= *max_port; p++ {
		go listener((*bind_host).String(), p, incoming)
	}

	go proxy(incoming)

	select {}
}
