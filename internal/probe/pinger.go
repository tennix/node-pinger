package probe

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tennix/node-pinger/internal/model"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type ipFamily string

const (
	familyIPv4 ipFamily = "ipv4"
	familyIPv6 ipFamily = "ipv6"
)

type probeKey struct {
	family ipFamily
	addr   string
	id     int
	seq    int
}

type probeReply struct{}

type Pinger struct {
	id      int
	seq     atomic.Uint32
	mu      sync.Mutex
	v4Write sync.Mutex
	v6Write sync.Mutex
	pending map[probeKey]chan probeReply
	v4      *icmp.PacketConn
	v6      *icmp.PacketConn
}

func NewPinger() (*Pinger, error) {
	p := &Pinger{
		id:      os.Getpid() & 0xffff,
		pending: make(map[probeKey]chan probeReply),
	}

	v4, v4Err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if v4Err == nil {
		p.v4 = v4
		go p.readLoop(v4, familyIPv4, 1)
	}

	v6, v6Err := icmp.ListenPacket("ip6:ipv6-icmp", "::")
	if v6Err == nil {
		p.v6 = v6
		go p.readLoop(v6, familyIPv6, 58)
	}

	if p.v4 == nil && p.v6 == nil {
		return nil, fmt.Errorf("open ICMP sockets: ipv4=%v ipv6=%v", v4Err, v6Err)
	}

	return p, nil
}

func (p *Pinger) Close() error {
	var joined error
	if p.v4 != nil {
		joined = errors.Join(joined, p.v4.Close())
	}
	if p.v6 != nil {
		joined = errors.Join(joined, p.v6.Close())
	}
	return joined
}

func (p *Pinger) Probe(peer model.Node, timeout time.Duration) (time.Duration, error) {
	ip := net.ParseIP(peer.InternalIP)
	if ip == nil {
		return 0, fmt.Errorf("invalid peer IP %q", peer.InternalIP)
	}

	family, conn, msgType, err := p.connectionForIP(ip)
	if err != nil {
		return 0, err
	}

	seq := int(p.seq.Add(1) & 0xffff)
	key := probeKey{family: family, addr: ip.String(), id: p.id, seq: seq}
	replyCh := make(chan probeReply, 1)
	p.storePending(key, replyCh)
	defer p.deletePending(key)

	message := &icmp.Message{
		Type: msgType,
		Code: 0,
		Body: &icmp.Echo{ID: p.id, Seq: seq, Data: []byte("node-pinger")},
	}
	packet, err := message.Marshal(nil)
	if err != nil {
		return 0, fmt.Errorf("marshal icmp message: %w", err)
	}

	deadline := time.Now().Add(timeout)
	writeMu := p.writeMutex(family)
	writeMu.Lock()
	defer writeMu.Unlock()
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return 0, fmt.Errorf("set write deadline: %w", err)
	}

	start := time.Now()
	if _, err := conn.WriteTo(packet, &net.IPAddr{IP: ip}); err != nil {
		return 0, fmt.Errorf("send icmp %s probe to %s: %w", family, peer.InternalIP, err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-replyCh:
		return time.Since(start), nil
	case <-timer.C:
		return 0, ErrTimeout
	}
}

func (p *Pinger) writeMutex(family ipFamily) *sync.Mutex {
	if family == familyIPv4 {
		return &p.v4Write
	}
	return &p.v6Write
}

var ErrTimeout = errors.New("probe timeout")

func (p *Pinger) connectionForIP(ip net.IP) (ipFamily, *icmp.PacketConn, icmp.Type, error) {
	if ip.To4() != nil {
		if p.v4 == nil {
			return "", nil, nil, fmt.Errorf("ipv4 probing unavailable")
		}
		return familyIPv4, p.v4, ipv4.ICMPTypeEcho, nil
	}
	if p.v6 == nil {
		return "", nil, nil, fmt.Errorf("ipv6 probing unavailable")
	}
	return familyIPv6, p.v6, ipv6.ICMPTypeEchoRequest, nil
}

func (p *Pinger) readLoop(conn *icmp.PacketConn, family ipFamily, protocol int) {
	buf := make([]byte, 1500)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return
		}
		n, peerAddr, err := conn.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return
		}

		msg, err := icmp.ParseMessage(protocol, buf[:n])
		if err != nil {
			continue
		}

		if !isEchoReplyType(family, msg.Type) {
			continue
		}

		echo, ok := msg.Body.(*icmp.Echo)
		if !ok {
			continue
		}

		ip := remoteIP(peerAddr)
		if ip == "" {
			continue
		}

		key := probeKey{family: family, addr: ip, id: echo.ID, seq: echo.Seq}
		p.mu.Lock()
		ch := p.pending[key]
		p.mu.Unlock()
		if ch == nil {
			continue
		}

		select {
		case ch <- probeReply{}:
		default:
		}
	}
}

func (p *Pinger) storePending(key probeKey, replyCh chan probeReply) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending[key] = replyCh
}

func (p *Pinger) deletePending(key probeKey) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pending, key)
}

func isEchoReplyType(family ipFamily, msgType icmp.Type) bool {
	if family == familyIPv4 {
		return msgType == ipv4.ICMPTypeEchoReply
	}
	return msgType == ipv6.ICMPTypeEchoReply
}

func remoteIP(addr net.Addr) string {
	switch typed := addr.(type) {
	case *net.IPAddr:
		if typed.IP != nil {
			return typed.IP.String()
		}
	case *net.UDPAddr:
		if typed.IP != nil {
			return typed.IP.String()
		}
	}
	return ""
}
