package lrmp

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/net/ipv4"
	"math/rand"
	"net"
	"strconv"
	"time"
)

type impl struct {
	stats       Stats
	domainStats DomainStats
	cxt         *Context
	idleTime    int64
	session     *msession
	ttl         int
	reports     map[Entity]*sender
	event       *timerTask
	nextTimeout time.Time
}

const maxPacketSize = MTU
const VersionNumber = 1
const Modulo32 int64 = int64(1) << 32
const checkInterval = 10000
const broadcastSrc = uint32(0xffffffff)

const (
	DATA_PT   = 0
	R_DATA_PT = 4
	U_DATA_PT = 8
	F_DATA_PT = 12
	NACK_PT   = 17
	R_NACK_PT = 18
	SR_PT     = 19
	RS_PT     = 20
	RR_PT     = 21
)

func newImpl(addr string, port int, ttl int, network string, profile Profile) (*impl, error) {

	group, err := net.ResolveUDPAddr("udp", addr+":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	ifi, err := net.InterfaceByName(network)
	if err != nil {
		return nil, err
	}

	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}

	var laddr net.IP

	for _, a := range addrs {
		if _, ok := a.(*net.IPNet); ok {
			laddr = a.(*net.IPNet).IP.To4()
			if laddr != nil {
				break
			}
		}
	}

	if laddr == nil {
		return nil, errors.New("interface does not have IP address")
	}

	l, err := net.ListenMulticastUDP("udp4", ifi, group)
	if err != nil {
		return nil, err
	}

	socket := ipv4.NewPacketConn(l)
	err = socket.SetMulticastInterface(ifi)
	if err != nil {
		return nil, err
	}
	err = socket.SetMulticastLoopback(true)
	if err != nil {
		return nil, err
	}

	cxt := newContext(laddr, ttl)

	impl := impl{ttl: ttl, cxt: cxt}
	impl.reports = make(map[Entity]*sender)

	impl.session = newSession(socket, &impl, group)

	impl.cxt.whoami = impl.cxt.sm.whoami

	impl.cxt.setProfile(&profile)

	impl.cxt.lrmp = &impl

	return &impl, nil
}

func (i *impl) startSession() {
	i.session.start()

	if i.cxt.recover == nil {
		i.initRecovery()
	}
}
func (i *impl) stopSession() {
	i.session.start()
}

func (i *impl) initRecovery() {
	if i.cxt.recover != nil {
		i.cxt.recover.stop()
	}

	i.cxt.recover = newRecovery(i.ttl, i.cxt)
}

func (i *impl) whoAmI() Entity {
	return i.cxt.whoami
}
func (i *impl) send(pack *Packet) error {
	if pack.reliable && i.cxt.whoami.lastTimeForData.IsZero() {
		i.sendSenderReport()
		i.cxt.whoami.initCache(i.cxt.profile.sendWindowSize)
		i.cxt.whoami.lastTimeForData = time.Now()
	}

	i.cxt.sender.enqueue(pack)

	if i.idleTime > 0 {
		i.idleTime = 0
		i.cxt.whoami.nextSRTime = time.Now().Add(time.Duration(i.cxt.senderReportInterval))

		i.startTimer(i.cxt.senderReportInterval)
	}
	return nil
}

func (i *impl) idle() {

	if isDebug() {
		logDebug("idle()")
	}

	now := time.Now()
	idleTime := i.cxt.sndInterval / 16

	if idleTime < 1000 {
		idleTime = 1000
	} else if idleTime > 4000 {
		idleTime = 4000
	}
	if i.event != nil {
		timer.recallTimer(i.event)
		i.event = nil
	}
	i.cxt.whoami.nextSRTime = addMillis(now, idleTime)
	i.startTimer(idleTime)
}

func (i *impl) startTimer(millis int) {
	t1 := addMillis(time.Now(), millis)

	if i.event != nil {
		if t1.After(i.nextTimeout) {
			return
		}

		timer.recallTimer(i.event)
	}
	if isDebug() {
		logDebug("next timeout in ", millis)
	}

	i.event = timer.registerTimer(millis, i, nil)
	i.nextTimeout = t1
}

func (i *impl) handleTimerTask(data interface{}, thetime time.Time) {
	i.event = nil

	p := NewPacket(false, 1024)

	p.scope = i.ttl
	p.offset = 0

	timeout := checkInterval

	cxt := i.cxt

	/*
	 * no sender reports if
	 * 1. never send data.
	 * 2. just sent out-of-band data.
	 * 3. has not sent data since the max silence (drop) time.
	 * send several sender reports when the transmission is stopped.
	 */
	if cxt.whoami.expected != cxt.whoami.startseq {
		if millis(thetime.Sub(cxt.whoami.lastTimeForData)) < sndDropTime {
			diff := int(millis(cxt.whoami.nextSRTime.Sub(thetime)))

			if diff <= 0 {
				p.appendSenderReport(cxt.whoami)

				cxt.stats.senderReports++

				/* update rate */

				octets := cxt.whoami.bytes - cxt.whoami.srBytes

				interval := millis(thetime.Sub(cxt.whoami.srTimestamp))

				interval = ((interval >> 8) * 1000) >> 8

				if interval > 0 {
					cxt.whoami.setRate(octets * 1000 / int(interval))
				}

				cxt.whoami.srBytes = cxt.whoami.bytes
				cxt.whoami.srPackets = cxt.whoami.packets
				cxt.whoami.srSeqno = cxt.whoami.expected
				cxt.whoami.srTimestamp = thetime

				if millis(thetime.Sub(cxt.whoami.lastTimeForData)) > i.idleTime {
					timeout = int(millis(thetime.Sub(cxt.whoami.lastTimeForData)))
					if timeout < 2000 {
						timeout = 2000
					}
				} else {
					timeout = cxt.senderReportInterval
				}

				cxt.whoami.nextSRTime = addMillis(thetime, timeout)
			} else {
				timeout = diff
			}

			if timeout > cxt.rcvReportSelInterval {
				timeout = cxt.rcvReportSelInterval
			}

			if cxt.profile.rcvReportSelection != NoReceiverReport && int(millis(thetime.Sub(cxt.whoami.rrSelectTime))) > cxt.rcvReportSelInterval {

				if cxt.stats.populationEstimate < cxt.sm.getNumberOfEntities() {
					cxt.stats.populationEstimate = cxt.sm.getNumberOfEntities()
				}

				cxt.whoami.rrInterval = 10 /* seconds */

				/*
				 * limit the number of reports to 100, so using the following
				 * formula probability*population < 100.
				 */
				cxt.whoami.rrProb = (100 << 16) / (cxt.stats.populationEstimate + 1)

				if cxt.whoami.rrProb > 0xffff {
					cxt.whoami.rrProb = 0xffff
				}

				p.appendRRSelection(cxt.whoami, cxt.whoami.rrProb, cxt.whoami.rrInterval)

				cxt.whoami.rrSelectTime = time.Now()
				cxt.whoami.rrReplies = 0
				cxt.stats.populationEstimate = 0
				cxt.stats.rrSelect++
			}
		}
		if isDebug() {
			logDebug("send sender report ", p.offset)
		}
	}

	/*
	 * check if send receiver reports.
	 */
	for e, s := range i.reports {

		delay := int(s.nextRRTime.Sub(thetime))

		if delay <= 0 {
			p.appendReceiverReport(s, cxt.whoami)

			cxt.stats.receiverReports++

			if s.rrProb > 0 { /* once */
				delete(i.reports, e)
			} else {
				delay = i.randomize(s.rrInterval)
				s.nextRRTime = addMillis(thetime, delay)
			}
		}
		if delay > 0 && delay < timeout {
			timeout = delay
		}
	}

	if p.offset > 0 {
		i.sendControlPacket(p, i.ttl)
	}

	/* prune the list of entities heard */
	cxt.sm.prune(rcvDropTime)
	i.startTimer(timeout)
}

func (i *impl) sendControlPacket(pack *Packet, ttl int) {
	cxt := i.cxt

	cxt.whoami.setLastTimeHeard(time.Now())

	cxt.stats.ctrlPackets++
	cxt.stats.ctrlBytes += int64(pack.offset)

	i.session.send(pack.buff, pack.offset, ttl)
}

func (i *impl) flush() {
	i.cxt.sender.flush()
}

func (i *impl) setTTL(ttl int) {
	i.ttl = ttl
	if i.cxt.recover != nil {
		i.cxt.recover.stop()
	}
	i.initRecovery()
}

/**
 * does 32-bit diff of seqno (seq1 - seq2). Handle overflow and underflow.
 * The result will be correct provided that the absolute diff is less than
 * Modulo32/2.
 */
func diff32(seq1, seq2 int64) int {
	diff := seq1 - seq2

	if diff > (Modulo32 >> 1) {
		diff -= Modulo32
	} else if diff < -(Modulo32 >> 1) {
		diff += Modulo32
	}

	return int(diff)
}

/* in interval 0.25i to 1.0i */
func (impl *impl) randomize(i int) int {
	return (int)(i * ((65536 + 3*(rand.Int()&0xffff)) >> 8) >> 10)
}

func (i *impl) sendSenderReport() {
	p := NewPacket(false, 64)

	p.scope = i.ttl
	p.offset = 0

	p.appendSenderReport(i.cxt.whoami)
	i.sendControlPacket(p, i.ttl)

	i.cxt.stats.senderReports++
}

func (i *impl) parse(buff []byte, totalLen int, ip net.IP) {

	cxt := i.cxt

	/*
	 * validity check.
	 */
	if totalLen < 12 {
		cxt.stats.badLength++

		if isDebug() {
			logDebug("packet too short (" + strconv.Itoa(totalLen) + ") " + ip.String())
		}

		return
	}

	v := int(buff[0]&0xff) >> 6

	if v != VersionNumber {
		cxt.stats.badVersion++

		if isDebug() {
			logDebug("incorrect version (" + strconv.Itoa(v) + ") " + ip.String())
		}
		return
	}

	/* entity ID */

	id := uint32(byteToInt(buff, 4))

	/* ignore loopback packets */

	if i.cxt.whoami.getID() == id && bytes.Equal(i.cxt.whoami.getAddress(), ip) {
		if isTrace() {
			logTrace("ignoring packet from me")
		}
		return
	}

	s := cxt.sm.lookup(id, ip)

	if s == nil {

		/*
		 * refused...
		 */
		logError("rejected packet from " + strconv.FormatInt(int64(id), 16) + "@" + ip.String())
		return
	}

	/* a rough estimate of distance XXX */

	d := int(buff[1] & 0xff)

	if s.getDistance() > d {
		s.setDistance(d)
	}

	/*
	 * loop to process all multiplexed packets.
	 */
	var offset = 0

	for offset < totalLen {
		len := int(byteToShort(buff, offset+2))

		if len < 12 || (len+offset) > totalLen {
			cxt.stats.badLength++

			logError("bad packet length " + strconv.Itoa(len))

			break
		}

		/* packet type */

		t := int(buff[offset] & 0x1f)

		if t >= 16 {
			cxt.stats.ctrlPackets++
			cxt.stats.ctrlBytes += int64(len)

			switch t {

			case NACK_PT:
				i.processNack(s, buff, offset, len)
				break

			case R_NACK_PT:
				i.processNackReply(s, buff, offset, len)
				break

			case SR_PT:
				i.processSenderReport(s, buff, offset, len)
				break

			case RS_PT:
				i.processRRSelection(s, buff, offset, len)
				break

			case RR_PT:
				i.processReceiverReport(s, buff, offset, len)
				break

			default:
				logError("bad control pt " + strconv.Itoa(t))
				break
			}
		} else {
			cxt.stats.dataPackets++
			cxt.stats.dataBytes += int64(len)

			b := make([]byte, totalLen)
			copy(b, buff)

			if t >= DATA_PT && t < R_DATA_PT {
				i.processData(s, b, offset, len)
			} else if t >= R_DATA_PT && t < U_DATA_PT {
				i.processRepairData(s, b, offset, len)
			} else if t >= U_DATA_PT && t < F_DATA_PT {
				i.processUnreliableData(s, b, offset, len)
			} else if t == R_DATA_PT {
				i.processFecData(s, b, offset, len)
			} else {
				logError("bad data pt " + strconv.Itoa(t))
			}
		}

		offset += len
	}

	s.setLastTimeHeard(time.Now())
}

func (i *impl) processNack(s Entity, buff []byte, offset int, len int) {
	scope := int(buff[offset+1] & 0xff)

	offset += 8

	timestamp := byteToInt(buff, offset)

	offset += 4
	len -= 12

	cxt := i.cxt

	for ; len >= 12; len -= 12 {

		/* the designated source */

		src := uint32(byteToInt(buff, offset))
		e := i.cxt.sm.get(src)

		if _, isSender := e.(*sender); !isSender {
			continue
		}

		offset += 4

		ev := newLossEvent(e.(*sender))

		ev.rcvSendTime = time.Now()
		ev.low = int64(byteToInt(buff, offset))
		offset += 4
		ev.bitmask = uint32(byteToInt(buff, offset))
		offset += 4
		ev.scope = scope
		ev.reporter = s
		ev.timestamp = timestamp

		if isDebug() {
			logDebug("got NACK " + fmt.Sprint(ev) + " @" + times(ev.rcvSendTime))
		}

		i.cxt.recover.processNack(ev)

		/* rate adaptation */

		if e == i.cxt.whoami {
			k := int(cxt.whoami.expected - int64(ev.low))

			if k > (cxt.whoami.cacheSize >> 1) {
				cxt.adjust = BigDecrease
			} else if k > (cxt.whoami.cacheSize / 3) {
				cxt.adjust = MediumDecrease
			} else if k > (cxt.whoami.cacheSize >> 2) {
				cxt.adjust = SmallDecrease
			} else {
				cxt.adjust = None
			}
		}
	}

	if len > 0 {
		cxt.stats.badLength++
	} else {
		s.incNack()
	}
}

func (i *impl) processNackReply(s Entity, buff []byte, offset int, len int) {
	scope := int(buff[offset+1])

	offset += 8
	len -= 8

	cxt := i.cxt

	for ; len >= 8; len -= 24 {
		to := uint32(byteToInt(buff, offset))

		offset += 4

		timestamp := byteToInt(buff, offset)

		offset += 4

		delay := byteToInt(buff, offset)

		offset += 4

		dataSrc := uint32(byteToInt(buff, offset))

		e := cxt.sm.get(dataSrc)

		if _, isSender := e.(*sender); !isSender {
			offset += 12
			continue
		}

		offset += 4

		ev := newLossEvent(e.(*sender))

		ev.rcvSendTime = time.Now()
		ev.reporter = cxt.sm.get(to)

		if ev.reporter == nil {
			offset += 8

			continue
		}

		ev.low = int64(byteToInt(buff, offset))
		offset += 4
		ev.bitmask = uint32(byteToInt(buff, offset))
		offset += 4
		ev.scope = scope
		ev.timestamp = timestamp

		cxt.recover.processNackReply(s, ev, delay)
	}

	if len > 0 {
		cxt.stats.badLength++
	}
}

func (i *impl) processSenderReport(e Entity, buff []byte, offset int, len int) {
	offset += 8

	timestamp := time.Unix(0, int64(byteToInt(buff, offset))*int64(time.Millisecond))

	offset += 4

	seqno := int64(byteToInt(buff, offset))

	offset += 4

	var s *sender

	cxt := i.cxt

	if _, isSender := e.(*sender); isSender {
		s = e.(*sender)
	} else {
		s = cxt.sm.lookupSender(e.getID(), e.getAddress(), seqno)
		s.setRate((cxt.profile.minRate + cxt.profile.maxRate) / 2)
	}

	s.srSeqno = seqno

	packets := byteToInt(buff, offset)

	offset += 4

	bytes := byteToInt(buff, offset)

	offset += 4

	if diff32(seqno, s.expected) > 0 {
		if diff32(seqno, s.maxseq) > 1 {
			s.maxseq = seqno - 1
		}

		cxt.recover.handleLoss(s)
	}

	/* estimate the rate */

	if !s.srTimestamp.IsZero() {
		interval := timestamp.Sub(s.srTimestamp)

		if interval > 0 {
			diff := bytes - s.srBytes

			if diff > 0 {
				s.setRate(int(float64(diff) / interval.Seconds()))
			}

			diff = packets - s.srPackets

			if diff > 0 {

				/*
				 * this is not the exact interval at which the sender is sending
				 * because of repairs.
				 */
				s.setInterval(int(float64(diff) / interval.Seconds()))
			}
		}
	}
	if isDebug() {
		logDebug("got SR ", e, " cur/next:", seqno, "/", s.expected, " rate:", s.rate)
	}

	s.srTimestamp = timestamp
	s.srPackets = packets
	s.srBytes = bytes
	cxt.stats.senderReports++
}

func (i *impl) processRRSelection(e Entity, buff []byte, offset int, len int) {
	cxt := i.cxt

	cxt.stats.rrSelect++

	if _, isSender := e.(*sender); !isSender {

		/*
		 * refused...
		 */
		if isDebug() {
			logDebug("receiver report sel from non sender")
		}

		return
	}

	s := e.(*sender)

	offset += 8
	s.rrTimestamp = byteToInt(buff, offset)
	offset += 4
	s.rrProb = byteToShort(buff, offset)
	offset += 2
	s.rrInterval = byteToShort(buff, offset)
	offset += 2
	s.rrInterval = s.rrInterval * 1000
	len -= 16

	for len >= 4 {
		id := uint32(byteToInt(buff, offset))

		if id == broadcastSrc || id == cxt.whoami.getID() {
			s.rrSelectTime = time.Now()
			s.rrReplies = 0

			send := true

			if s.rrProb > 0 {
				i := rand.Int()
				i &= 0xffff
				if i > s.rrProb {
					send = false
				}
			} else if s.rrInterval == 0 {
				send = false
			}
			if isDebug() {
				logDebug("RR select prob=", s.rrProb, " interv=", s.rrInterval, " ", send)
			}
			if send {
				now := time.Now()

				delay := i.randomize(s.rrInterval)

				if s.nextRRTime.After(now) {
					break // if we keep rescheduling we never send
				}

				s.nextRRTime = time.Now().Add(time.Duration(delay) * time.Millisecond)

				i.startTimer(delay)

				_, ok := i.reports[s]

				if !ok {
					i.reports[s] = s
				}
			} else {
				delete(i.reports, s)
			}

			break
		}

		offset += 4
		len -= 4
	}
}

func (i *impl) processReceiverReport(e Entity, buff []byte, offset int, len int) {
	scope := int(buff[offset+1])

	offset += 8
	len -= 8

	now := time.Now()

	cxt := i.cxt

	for len >= 20 {
		cxt.stats.receiverReports++

		to := uint32(byteToInt(buff, offset))

		offset += 4

		s := cxt.sm.get(to)

		if _, isSender := s.(*sender); isSender {
			sender := s.(*sender)
			timestamp := byteToInt(buff, offset)

			offset += 4

			/* maybe we missed RRSelect packet, ignore */

			if timestamp == sender.rrTimestamp {
				sender.rrReplies++

				/* suppose the estimation scheme is not changed */

				if sender.rrProb > 0 {
					cxt.stats.populationEstimate = (sender.rrReplies<<16)/sender.rrProb + 1
					cxt.stats.populationEstimateTime = now
				}
			} else {
				if sender != cxt.whoami {
					sender.rrReplies = 0
					cxt.stats.populationEstimate = 0
				}
			}
			if s == cxt.whoami {
				delay := byteToInt(buff, offset)

				/* NTP offset is subtracted */

				rtt := ntp32(now.UnixNano()/int64(time.Millisecond)) - timestamp - delay

				rtt = fixedPoint32ToMillis(rtt)

				if rtt >= 0 {
					e.setRTT(rtt)

					d := cxt.recover.lookupDomain(scope)

					if d != nil {
						d.updateMRTT(rtt)
					}
				} else {
					logError("bad rtt ", rtt, " ", e, " ", ntp32(nowMillis()), "/", delay, "/", timestamp)
				}
				if isDebug() {
					logDebug("RR from ", e, " rtt=", rtt)
				}
			}

			/* other field ignored */

			offset += 12
		} else {
			offset += 16
		}

		len -= 20
	}
}

/* process DATA packet */
func (i *impl) processData(from Entity, buff []byte, offset int, len int) {
	seqno := int64(byteToInt(buff, offset+12))

	/*
	 * check the source.
	 */
	var source *sender

	cxt := i.cxt

	if _, isSender := from.(*sender); !isSender {
		source = cxt.sm.lookupSender(from.getID(), from.getAddress(), seqno)
	} else {
		source = from.(*sender)
	}

	/*
	 * check the seqno. Ignore duplicate.
	 */
	diff := diff32(seqno, source.expected)

	if diff < 0 {
		source.incDuplicate()

		return
	}

	pack := source.getPacket(seqno)

	if pack != nil {
		source.incDuplicate()

		pack.scope = int(buff[offset+1] & 0xff)
		pack.rcvSendTime = time.Now()
	}

	/* pack the data into a packet */

	pack = newDataPacket(true, buff, offset, len)
	pack.seqno = seqno
	pack.retransmit = false
	pack.sender = from
	pack.source = source

	/*
	 * update stats.
	 */
	source.lastTimeForData = pack.rcvSendTime

	source.updateJitter(byteToInt(buff, offset+8))

	source.lastseq = seqno

	source.incPackets()
	source.incBytes(pack.datalen)

	if isDebug() {
		logDebug("data/exp:", seqno, "/", source.expected, " @", times(pack.rcvSendTime), " /", pack.scope)
	}
	if pack.seqno > source.maxseq {
		source.maxseq = pack.seqno
	}

	/*
	 * check sequence number.
	 */
	if diff == 0 {

		/*
		 * In order, keeps a local copy in cache for local repair.
		 */
		if cxt.profile.sendRepair {
			source.putPacket(pack)
		}

		/*
		 * deliver all cached packets in sequence.
		 */
		for pack != nil {
			source.incExpected()
			i.deliverData(pack)

			pack = source.getPacket(source.expected)
		}
	} else if diff <= source.cacheSize {

		/*
		 * out of order but in range, that is, loss is still recoverable.
		 * cache the packet and process loss.
		 */
		source.putPacket(pack)
		cxt.recover.handleLoss(source)
	} else {

		/*
		 * out of range: diff > maxCacheSize.
		 * we have missed too much, maybe the sender is sending data too
		 * fast or we have a network trouble.
		 * Could not sync with the transmission. Reset the data stream and try
		 * to catch up.
		 */
		i.handleSyncError(source, BufferOverrun)

		if cxt.profile.sendRepair {
			source.putPacket(pack)
		}
	}
}

/**
 * handles a reception failure event.
 */
func (i *impl) handleSyncError(s *sender, cause int) {
	logError("reception failure @", s.expected, "/", s.maxseq, " cause=", cause)

	/* for continuous losses, we should report only one event */

	var ev *errorEvent

	if s.expected != (s.lastError + 1) {
		ev = newErrorEvent()
		ev.source = s
		ev.loser = i.cxt.whoami
		ev.cause = cause
		ev.seqlost = int(s.expected)
		i.cxt.stats.failures++

		if i.cxt.profile.Handler != nil {
			i.cxt.profile.Handler.ProcessEvent(UNRECOVERABLE_SEQUENCE_ERROR, ev)
		}
	}

	s.lastError = s.expected

	diff := diff32(s.maxseq, s.expected)

	if diff < 0 {
		s.clearCache(s.maxseq)
	} else {
		s.incExpected()

		diff--

		/* deliver in order packets */

		var lastpack *Packet

		for diff > s.cacheSize {
			pack := s.getPacket(s.expected)

			if pack != nil {
				i.deliverData(pack)
			} else {
				if lastpack != nil {
					if i.cxt.profile.Handler != nil {
						if ev == nil {
							ev = newErrorEvent()
							ev.source = s
							ev.loser = i.cxt.whoami
							ev.cause = cause
						}

						s.lastError = s.expected
						ev.seqlost = int(s.expected)
						i.cxt.stats.failures++

						i.cxt.profile.Handler.ProcessEvent(UNRECOVERABLE_SEQUENCE_ERROR, ev)
					}
				}
			}

			lastpack = pack

			s.incExpected()

			diff--
		}
	}

	/* deliver in order packets */

	for {
		pack := s.getPacket(s.expected)

		if pack == nil {
			break
		}

		i.deliverData(pack)
		s.incExpected()
	}

	if isDebug() {
		logDebug("synced to ", s.expected)
	}

	ev1 := i.cxt.recover.lookup(s, i.cxt.whoami)

	if ev1 != nil {

		/* update the low seqno to indicate that this is not repaired */

		ev1.low = s.expected
		ev1.nextAction = SendNack

		if ev1.nackCount > 1 {
			ev1.nackCount--
		}
	} else {
		i.cxt.recover.handleLoss(s)
	}
}

func (i *impl) deliverData(pack *Packet) {
	if pack.reliable {
		if isDebug() {
			logDebug("deliver #", pack.seqno, " len=", pack.datalen, "from", pack.source)
		}

		/*
		 * remove from cache if don't participate in local recovery.
		 */
		if !i.cxt.profile.sendRepair {
			pack.source.(*sender).removePacket(pack)
		}
	} else if isDebug() {
		logDebug("deliver out-of-band", " len=", pack.datalen)
	}
	if i.cxt.profile.Handler != nil {
		i.cxt.profile.Handler.ProcessData(pack)
	}
}

/* process R_DATA packet */

func (i *impl) processRepairData(from Entity, buff []byte, offset int, len int) {

	cxt := i.cxt
	/*
	 * check the source.
	 */
	e := cxt.sm.get(uint32(byteToInt(buff, offset+8)))

	if _, isSender := e.(*sender); !isSender {

		/* have never heard from the sender */

		return
	}

	source := e.(*sender)

	source.incRepairs()

	/*
	 * check the seqno. Ignore duplicate, update reception stats and stop.
	 */
	seqno := int64(byteToInt(buff, offset+12))
	pack := source.getPacket(seqno)

	if pack != nil {
		pack.scope = int(buff[offset+1] & 0xff)
		pack.rcvSendTime = time.Now()

		if source != cxt.whoami {
			source.incDuplicate()

			/* call this method for possible duplicate repair suppression */

			cxt.recover.heardRepair(pack, true)
		}

		return
	} else if source == cxt.whoami {

		/*
		 * ignore repair packets for my own.
		 */
		return
	}

	diff := diff32(seqno, source.expected)

	if diff < 0 {
		source.incDuplicate()

		return
	}

	/*
	 * at this point it is really a repair.
	 */
	pack = newDataPacket(true, buff, offset, len)
	pack.retransmit = true
	pack.seqno = seqno
	pack.source = source
	pack.sender = from

	/* call this method for update stats */

	cxt.recover.heardRepair(pack, false)

	/*
	 * update stats.
	 */
	if pack.sender == source {
		source.lastTimeForData = pack.rcvSendTime
		source.lastseq = pack.seqno
	}

	source.incPackets()
	source.incBytes(pack.datalen)

	if isDebug() {
		logDebug("repair/exp:", pack.seqno, "/", source.expected, " @", times(pack.rcvSendTime), " /", pack.scope)
	}
	if pack.seqno > source.maxseq {
		source.maxseq = pack.seqno
	}

	/*
	 * further check.
	 */
	if diff == 0 {

		/*
		 * good repair, keeps a local copy in cache for local repair.
		 */
		if cxt.profile.sendRepair {
			source.putPacket(pack)
		}

		/*
		 * deliver all cached packets in sequence.
		 */
		for pack != nil {
			source.incExpected()
			i.deliverData(pack)

			pack = source.getPacket(source.expected)
		}
	} else if diff <= source.cacheSize {

		/*
		 * out of order but in range, that is, loss is still recoverable.
		 * cache the packet and process loss.
		 */
		source.putPacket(pack)
		cxt.recover.handleLoss(source)
	}

	/*
	 * out of range: diff > maxCacheSize. Ignore.
	 */
}

/* process U_DATA packet */
func (i *impl) processUnreliableData(from Entity, buff []byte, offset int, len int) {
	i.cxt.stats.outOfBand++

	/* pack the data into a packet */

	pack := newDataPacket(false, buff, offset, len)

	pack.source = from

	i.deliverData(pack)

	return
}

/* process F_DATA packet */
func (i *impl) processFecData(from Entity, buff []byte, offset int, len int) {
}
func (i *impl) sendDataPacket(pack *Packet, resend bool) {
	len := pack.formatDataPacket(resend)

	i.session.send(pack.buff, len, pack.scope)

	if resend {
		d := i.cxt.recover.lookupDomain(pack.scope)

		d.stats.repairPackets++
		d.stats.repairBytes += int64(len)

		i.cxt.whoami.incRepairs()
	}

	pack.rcvSendTime = time.Now()

	if pack.reliable { /* XXXXXXXXX */
		i.cxt.whoami.lastTimeForData = pack.rcvSendTime
	}

	i.cxt.whoami.setLastTimeHeard(pack.rcvSendTime)
	i.cxt.whoami.incPackets()
	i.cxt.whoami.incBytes(len)

	i.cxt.stats.dataPackets++
	i.cxt.stats.dataBytes += int64(len)
}
