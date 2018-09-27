package lrmp

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"math/rand"
	"net"
)

type msession struct {
	socket  *ipv4.PacketConn
	impl    *impl
	packets int
	bytes   int64
	gaddr   *net.UDPAddr
}

// enable the following to test recovery on reliable networks
const DropPackets = true

func newSession(socket *ipv4.PacketConn, impl *impl, gaddr *net.UDPAddr) *msession {
	s := msession{socket: socket, impl: impl, gaddr: gaddr}
	return &s
}

func (s *msession) start() {
	go func() {
		var buffer [maxPacketSize]byte
		for {
			n, _, addr, err := s.socket.ReadFrom(buffer[:])

			if DropPackets && drop() {
				//Logger.trace(this, "drop packet")
				continue
			}

			s.packets += 1
			s.bytes += int64(n)

			s.impl.parse(buffer[:n], n, addr.(*net.UDPAddr).IP)

			if err != nil {
				fmt.Println("reader existing")
				break
			}
		}
	}()
}
func (s *msession) stop() {
	s.socket.Close()
}

/**
 * sends data to the session using the provided TTL.
 */
func (s *msession) send(buf []byte, len int, ttl int) {
	if DropPackets && drop() {
		logDebug("drop packet")
		return
	}

	s.socket.SetMulticastTTL(ttl)
	s.socket.SetTTL(ttl)

	_, err := s.socket.WriteTo(buf[:len], nil, s.gaddr)
	if err != nil {
		logError("unable to write to socket", err)
	}
}

/*
 * just for simulation.
 */
func drop() bool {
	if rand.Intn(10) < 2 {
		return true
	} else {
		return false
	}
}
