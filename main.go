package main

import (
	"fmt"
	ka "github.com/felixge/tcpkeepalive"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
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
	keepalive          = kingpin.Flag("keepalive", "enable keepalive").Short('k').Bool()
	keepalive_interval = kingpin.Flag("keepalive-interval", "time between keepalive packets").Short('K').Default("1").Int()
	keepalive_count    = kingpin.Flag("keepalive-count", "number of failed probes before dropping the connection").Short('C').Default("30").Int()
	keepalive_idle     = kingpin.Flag("keepalive-idle", "idle time before starting keepalive probes").Short('I').Default("5").Int()
	destination_host   = kingpin.Flag("destination-host", "Destination host").Short('d').Default("127.0.0.1").IP()
	bind_host          = kingpin.Flag("bind", "Bind (listen) host").Short('b').Default("0.0.0.0").IP()
	min_port           = kingpin.Flag("min-port", "minimum port value").Short('m').Default("1").Int64()
	max_port           = kingpin.Flag("max-port", "maximum port value").Short('M').Default("65535").Int64()
	port_list          = kingpin.Flag("ports", "comma separated list of ports to use").Default("").Short('p').String()
	connections        map[int64]*net.TCPConn
	incoming           chan *Incoming
)

func init() {
	connections = make(map[int64]*net.TCPConn)
	// more incoming connections than the number of available ports
	incoming = make(chan *Incoming, 100000)
	kingpin.CommandLine.HelpFlag.Short('h')
}

func listener(host string, port int64, c chan *Incoming) {
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
		log.Printf("incoming connection from %s", conn.RemoteAddr().String())
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
			log.Printf("NEW cached connection (keepalive: %t) to %s from %s", *keepalive, incoming.Conn.LocalAddr().String(), incoming.Conn.RemoteAddr().String())
			if *keepalive {
				err = ka.SetKeepAlive(out, time.Duration(*keepalive_idle)*time.Second, *keepalive_count, time.Duration(*keepalive_interval)*time.Second)
				if err != nil {
					log.Printf("cannot enable tcp keepalive for connection, using standard connection")
				}
			}
			connections[incoming.Port] = out
		} else {
			log.Printf("REUSING cached connection to %s from %s", incoming.Conn.LocalAddr().String(), incoming.Conn.RemoteAddr().String())
		}
		// start the copy on connection until close from source
		go broker(out, incoming.Conn)
		go broker(incoming.Conn, out)
	}
}

func split_ports(p string) (ports []int64) {
	p_list := strings.Split(p, ",")
	for _, port := range p_list {
		val, err := strconv.ParseInt(port, 10, 64)
		if err != nil {
			panic(err)
		}
		ports = append(ports, val)
	}
	return ports
}

func main() {
	kingpin.Parse()
	log.Printf("sending incoming connections to %s", *destination_host)

	if *port_list != "" {
		ports := split_ports(*port_list)
		for _, p := range ports {
			go listener((*bind_host).String(), p, incoming)
		}
	} else {
		for p := *min_port; p <= *max_port; p++ {
			go listener((*bind_host).String(), p, incoming)
		}
	}

	go proxy(incoming)

	select {}
}
