package lrmp

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"math/rand"
	"net"
)

type msession struct {
	con     *ipv4.PacketConn
	impl    *impl
	packets int
	bytes   int64
	addr    *net.UDPAddr
}

func newSession(con *ipv4.PacketConn, impl *impl, addr *net.UDPAddr) *msession {
	s := msession{con: con, impl: impl, addr: addr}
	return &s
}

func (s *msession) start() {
	go func() {
		var buffer [maxPacketSize]byte
		for {
			n, _, addr, err := s.con.ReadFrom(buffer[:])

			if false && drop() {
				//Logger.trace(this, "drop packet")
				continue
			}

			s.packets += 1
			s.bytes += int64(n)

			s.impl.parse(buffer[:n], n, addr.(*net.UDPAddr))

			if err != nil {
				fmt.Println("reader existing")
				break
			}
		}
	}()
}
func (s *msession) stop() {
	s.con.Close()
}

/**
 * sends data to the session using the provided TTL.
 */
func (s *msession) send(buf []byte, len int, ttl int) {
	if false && drop() {
		logDebug("drop packet")
		return
	}

	_, err := s.con.WriteTo(buf[:len], nil, s.addr)
	if err != nil {
		logError("unable to write to socket", err)
	}
}

/*
 * just for simulation.
 */
var r = rand.Rand{}

func drop() bool {
	if r.Intn(10) < 2 {
		return true
	} else {
		return false
	}
}
