package lrmp

import (
	"math/rand"
	"net"
	"strconv"
	"time"
)

const rcvDropTime = 60000
const sndDropTime = 600000
const maxSrc = 128

type Entity interface {
	getID() int
	getAddress() *net.UDPAddr
	setLastTimeHeard(time time.Time)
	getLastTimeHeard() time.Time
	setAddress(addr *net.UDPAddr)
	reset()
	setID(id int)
	getDistance() int
	setDistance(distance int)
	incNack()
	setRTT(rtt int)
	getRTT() int
}

type EntityImpl struct {
	ipAddr        *net.UDPAddr
	lastTimeHeard time.Time
	nack          int
	id            int

	// round trip time in millis.

	rtt int
	// approx number of hops from local site.
	distance int
}

func (e *EntityImpl) String() string {
	return strconv.FormatInt(int64(e.getID()), 16) + "@" + e.getAddress().String()
}

func (e *EntityImpl) getID() int {
	return e.id
}

func (e *EntityImpl) setID(id int) {
	e.id = id
}

func (e *EntityImpl) incNack() {
	e.nack++
}

func (e *EntityImpl) getAddress() *net.UDPAddr {
	return e.ipAddr
}
func (e *EntityImpl) setLastTimeHeard(time time.Time) {
	e.lastTimeHeard = time
}
func (e *EntityImpl) getLastTimeHeard() time.Time {
	return e.lastTimeHeard
}
func (e *EntityImpl) setAddress(addr *net.UDPAddr) {
	e.ipAddr = addr
}
func (e *EntityImpl) getRTT() int {
	return e.rtt
}
func (e *EntityImpl) setRTT(rtt int) {
	e.rtt = rtt
}
func (e *EntityImpl) reset() {
	e.nack = 0
	e.lastTimeHeard = time.Unix(0, 0)
	e.distance = 255
}

func (e *EntityImpl) getDistance() int {
	return e.distance
}
func (e *EntityImpl) setDistance(distance int) {
	e.distance = distance
}

type entityManager struct {
	entities map[int]Entity
	whoami   *sender
	profile  *Profile
}

func newEntityManager(addr net.Addr) *entityManager {
	i := allocateID()

	var initSeqno int64 = 0

	for initSeqno <= 0 {
		initSeqno = int64(rand.Int() & 0xffff)
	}

	em := entityManager{entities: make(map[int]Entity)}

	em.whoami = newSender(i, addr.(*net.UDPAddr), initSeqno)

	if isDebug() {
		logDebug("local user=", em.whoami, " seqno=", em.whoami.expected)
	}

	em.add(em.whoami)

	return &em
}

func allocateID() int {
	return rand.Int()
}

func (m *entityManager) lookup(srcId int, addr *net.UDPAddr) Entity {
	s := m.entities[srcId]

	if s != nil {
		if s.getAddress().String() != addr.String() {
			_, ok := s.(*sender)
			if ok {
				return nil // if the registered is a sender, reject new one
			}

			silence := time.Now().Sub(s.getLastTimeHeard())

			if silence < time.Duration(rcvDropTime)*time.Millisecond {
				return nil
			}

			s.setAddress(addr)
			s.reset()

			return s
		} else {
			return s
		}
	}

	/*
	 * find duplicate, i.e., at the same net address, because an entity
	 * may rejoined the session.
	 */

	for _, e := range m.entities {
		_, isSender := e.(*sender)

		if e != m.whoami && !isSender {
			if e.getAddress().String() == addr.String() {
				silence := time.Now().Sub(e.getLastTimeHeard())

				if silence >= time.Duration(rcvDropTime)*time.Millisecond {
					m.remove(e)
					e.setID(srcId)
					m.add(e)
					e.reset()

					return e
				}
			}
		}
	}

	s = &EntityImpl{}
	s.setID(srcId)
	s.setAddress(addr)

	m.add(s)

	return s
}

func (m *entityManager) remove(e Entity) {
	if e != m.whoami {
		delete(m.entities, e.getID())

		if _, isSender := e.(*sender); isSender {
			if m.profile.Handler != nil {
				m.profile.Handler.ProcessEvent(END_OF_SEQUENCE, e)
			}
		}
	}
}
func (m *entityManager) add(e Entity) {
	if len(m.entities) > maxSrc {
		for maxSilence := rcvDropTime; len(m.entities) > maxSrc; {
			m.prune(maxSilence)

			if maxSilence > 10000 {
				maxSilence -= 10000
			} else {
				break
			}
		}
	}

	m.entities[e.getID()] = e
}

func (m *entityManager) prune(maxSilence int) {
	now := time.Now()

	for _, e := range m.entities {
		if e != m.whoami {
			silence := now.Sub(e.getLastTimeHeard())
			if silence >= time.Duration(sndDropTime)*time.Millisecond {
				delete(m.entities, e.getID())
			} else if _, isSender := e.(*sender); !isSender && silence >= time.Duration(maxSilence)*time.Millisecond {
				delete(m.entities, e.getID())
			}
		}
	}
}
func (m *entityManager) get(srcId int) Entity {
	return m.entities[srcId]
}

/* checks entity table to find one matching src id for data packets,
* provided that prior to any data reception LRMP control packets should
* be heard first.
 */
func (m *entityManager) demux(srcId int, netaddr *net.UDPAddr) Entity {
	s := m.entities[srcId]

	if s == nil {
		return nil
	}
	if s.getAddress().String() != netaddr.String() {
		return nil
	}

	return s
}

func (m *entityManager) lookupSender(srcId int, netaddr *net.UDPAddr, seqno int64) *sender {
	var s *sender

	e := m.demux(srcId, netaddr)

	if e == nil {
		s = newSender(srcId, netaddr, seqno)
		s.initCache(m.profile.rcvWindowSize)
		m.add(s)
	} else if _, isSender := e.(*sender); !isSender {
		s = newSender(srcId, netaddr, seqno)
		s.initCache(m.profile.rcvWindowSize)
		m.remove(e)
		m.add(s)
	} else {
		s = e.(*sender)
	}

	return s
}
func (m *entityManager) getNumberOfEntities() int {
	return len(m.entities)
}
