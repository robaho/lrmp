package lrmp

import (
	"bytes"
	"math/rand"
	"net"
	"strconv"
	"time"
)

const rcvDropTime = 60000
const sndDropTime = 600000
const maxSrc = 128

type Entity interface {
	getID() uint32
	getAddress() net.IP
	setLastTimeHeard(time time.Time)
	getLastTimeHeard() time.Time
	setAddress(ip net.IP)
	reset()
	setID(id uint32)
	getDistance() int
	setDistance(distance int)
	incNack()
	setRTT(rtt int)
	getRTT() int
}

type EntityImpl struct {
	ipAddr        net.IP
	lastTimeHeard time.Time
	nack          int
	id            uint32

	// round trip time in millis.

	rtt int
	// approx number of hops from local site.
	distance int
}

func (e *EntityImpl) String() string {
	return strconv.FormatInt(int64(e.getID()), 16) + "@" + e.getAddress().String()
}

func (e *EntityImpl) getID() uint32 {
	return e.id
}

func (e *EntityImpl) setID(id uint32) {
	e.id = id
}

func (e *EntityImpl) incNack() {
	e.nack++
}

func (e *EntityImpl) getAddress() net.IP {
	return e.ipAddr
}
func (e *EntityImpl) setLastTimeHeard(time time.Time) {
	e.lastTimeHeard = time
}
func (e *EntityImpl) getLastTimeHeard() time.Time {
	return e.lastTimeHeard
}
func (e *EntityImpl) setAddress(ip net.IP) {
	e.ipAddr = ip
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
	entities map[uint32]Entity
	whoami   *sender
	profile  *Profile
}

func newEntityManager(ip net.IP) *entityManager {
	i := allocateID()

	var initSeqno int64 = 0

	for initSeqno <= 0 {
		initSeqno = int64(rand.Int() & 0xffff)
	}

	em := entityManager{entities: make(map[uint32]Entity)}

	em.whoami = newSender(i, ip, initSeqno)

	if isDebug() {
		logDebug("local user=", em.whoami, " seqno=", em.whoami.expected)
	}

	em.add(em.whoami)

	return &em
}

func allocateID() uint32 {
	return uint32(rand.Int())
}

func (m *entityManager) lookup(srcId uint32, ip net.IP) Entity {
	s := m.entities[srcId]

	if s != nil {
		if !bytes.Equal(s.getAddress(), ip) {
			_, ok := s.(*sender)
			if ok {
				return nil // if the registered is a sender, reject new one
			}

			silence := time.Now().Sub(s.getLastTimeHeard())

			if silence < time.Duration(rcvDropTime)*time.Millisecond {
				return nil
			}

			s.setAddress(ip)
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
			if bytes.Equal(e.getAddress(), ip) {
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
	s.setAddress(ip)

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
func (m *entityManager) get(srcId uint32) Entity {
	return m.entities[srcId]
}

/* checks entity table to find one matching src id for data packets,
* provided that prior to any data reception LRMP control packets should
* be heard first.
 */
func (m *entityManager) demux(srcId uint32, ip net.IP) Entity {
	s := m.entities[srcId]

	if s == nil {
		return nil
	}
	if !bytes.Equal(s.getAddress(), ip) {
		return nil
	}

	return s
}

func (m *entityManager) lookupSender(srcId uint32, ip net.IP, seqno int64) *sender {
	var s *sender

	e := m.demux(srcId, ip)

	if e == nil {
		s = newSender(srcId, ip, seqno)
		s.initCache(m.profile.rcvWindowSize)
		m.add(s)
	} else if _, isSender := e.(*sender); !isSender {
		s = newSender(srcId, ip, seqno)
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
