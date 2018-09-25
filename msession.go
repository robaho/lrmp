package lrmp

import (
	"fmt"
	"math/rand"
	"net"
)

type msession struct {
	con     *net.UDPConn
	impl    *impl
	packets int
	bytes   int64
}

func newSession(con *net.UDPConn, impl *impl) *msession {
	s := msession{con: con, impl: impl}
	return &s
}

func (s *msession) start() {
	go func() {
		var buffer [maxPacketSize]byte
		for {
			n, addr, err := s.con.ReadFromUDP(buffer[:])

			if false && drop() {
				//Logger.trace(this, "drop packet")
				continue
			}

			s.packets += 1
			s.bytes += int64(n)

			s.impl.parse(buffer[:n], n, addr)

			fmt.Println("I got a packet from ", addr)
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
		//Logger.trace(this, "drop packet");
		return
	}

	s.con.WriteTo(buf, s.con.RemoteAddr())
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