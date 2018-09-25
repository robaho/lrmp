package lrmp

import (
	"net"
	"time"
)

type sender struct {
	EntityImpl
	lastTimeForData time.Time
	nextSRTime      time.Time
	cache           packetCache
	cacheSize       int
	startseq        int64
	maxseq          int64
	expected        int64
	lastseq         int64
	rrAbsLost       int64
	rrMaxSeqno      int64
	lastError       int64
	packets         int
	bytes           int
	rate            int
	interval        int
	transit         int
	jitter          int
	srTimestamp     time.Time
	nextRRTime      time.Time
	duplicates      int
	repairs         int
	drops           int
	srSeqno         int64
	srBytes         int
	srPackets       int
	rrTimestamp     int
	rrProb          int
	rrInterval      int
	rrSelectTime    time.Time
	rrReplies       int
}

func newSender(id int, netaddr *net.UDPAddr, start int64) *sender {
	s := sender{}
	s.id = id
	s.ipAddr = netaddr
	s.initCache(128)
	s.resetWithSeqNo(start)

	return &s
}

func (s *sender) resetWithSeqNo(initialSeqno int64) {

	s.reset()

	s.lastError = 0
	s.packets = 0
	s.bytes = 0
	s.rate = 0

	/* default to 1 kilo byte packets at 128 kbps */

	s.interval = 64
	s.transit = 0
	s.jitter = 0
	s.lastTimeForData = time.Unix(0, 0)
	s.srTimestamp = time.Unix(0, 0)
	s.nextSRTime = time.Unix(0, 0)
	s.nextRRTime = time.Unix(0, 0)
	s.duplicates = 0
	s.repairs = 0
	s.drops = 0

	s.clearCache(initialSeqno)
}

func (s *sender) initCache(cacheSize int) {
	s.cache = newPacketCache(cacheSize)
	s.cacheSize = s.cache.getMaxSize()
}

func (s *sender) clearCache(initialSeqno int64) {
	s.startseq = initialSeqno
	s.maxseq = initialSeqno - 1
	s.expected = initialSeqno
	s.lastseq = s.maxseq
	s.rrAbsLost = 0
	s.rrMaxSeqno = s.maxseq

	s.cache.clear()
}
func (s *sender) setRate(rate int) {
	s.rate = rate
}
func (s *sender) setInterval(interval int) {
	s.interval = interval
}
func (s *sender) incDuplicate() {
	s.duplicates++
}
func (s *sender) getPacket(seqno int64) *Packet {
	return s.cache.getPacket(seqno)
}
func (s *sender) updateJitter(timestamp int) {

}
func (s *sender) incPackets() {
	s.packets++
}
func (s *sender) incBytes(bytes int) {
	s.bytes += bytes
}
func (s *sender) putPacket(packet *Packet) {
	s.cache.addPacket(packet)
}
func (s *sender) incExpected() {
	s.expected++
}
func (s *sender) removePacket(packet *Packet) {
	s.cache.removePacket(packet)
}
func (s *sender) isCached(seqno int64) bool {
	return s.cache.containPacket(seqno)

}
