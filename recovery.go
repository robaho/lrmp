package lrmp

import (
	"math/rand"
	"time"
)

type recovery struct {
	cxt    *Context
	ttl    int
	domain *domain
	random rand.Rand
	task   *timerTask
	dummy  *Packet
}

const MaxTries = 8

func (r *recovery) handleTimerTask(data interface{}, thetime time.Time) {
	r.task = nil

	if isDebug() {
		logDebug("recovery handle timeout: ", r.domain.lossTab.Len())
	}

	var ev *lossEvent

	for elem := r.domain.lossTab.Front(); elem != nil; elem = elem.Next() {
		ev = elem.Value.(*lossEvent)

		if ev.timeoutTime.After(thetime) {
			continue
		}

		/*
		 * first process loss events reported by remote sites.
		 */
		if ev.reporter != r.cxt.whoami {
			r.domain.lossTab.remove(ev)
			r.resend(ev)

			continue
		}

		/*
		 * process loss events occurred at local site,
		 * check if need to send NACK.
		 */
		s := ev.source

		/* update loss event */

		ev.computeBitmask()

		/* check for stop */

		if ev.low < 0 {

			/*
			 * all lost packets repaired.
			 */
			if isDebug() {
				logDebug("Bravo!")
			}

			r.domain.lossTab.Remove(elem)

			continue
		} else if s.lost {
			r.cxt.lrmp.handleSyncError(ev.source, SenderGone)
			r.domain.lossTab.Remove(elem)
			continue
		} else if ev.nackCount >= MaxTries {
			r.cxt.lrmp.handleSyncError(ev.source, MaxTriesReached)
			r.domain.lossTab.Remove(elem)
			continue
		}

		/*
		 * If one of the following events happended,
		 * 1. a NACK has been received.
		 * 2. a NACK reply has been received.
		 * 3. one or more repair packets have been received.
		 * Don't send NACK and don't delete the event since the reporter
		 * or responder may leave. Schedule another timer to keep the
		 * repair process active.
		 */
		switch ev.nextAction {

		case DelayAndStay:

			/* just schedule the next timer */

			ev.nackCount++

			r.nackTimer(ev)

			ev.nextAction = SendNack

			break

		case DelayAndGoDown:
			r.goDown(ev)

			ev.nextAction = SendNack

			break

		case SendNack:

			/*
			 * This is the case where nothing has happened during the timer
			 * schedule period for this loss. So send a NACK.
			 * One try for lower domains and MaxTries for all domains.
			 */
			if r.dummy == nil {
				r.dummy = NewPacket(false, 64)
				r.dummy.sender = r.cxt.whoami
			}

			r.dummy.scope = ev.scope
			r.dummy.offset = 0

			r.dummy.appendNack(ev)

			if isDebug() {
				logDebug("send NACK ", ev)
			}

			r.cxt.lrmp.sendControlPacket(r.dummy, ev.scope)

			ev.domain.stats.nack++
			ev.domain.failedNack++
			ev.rcvSendTime = thetime

			r.cxt.whoami.incNack()
			r.goUp(ev)

			break

		case DelayAndGoUp:
			r.goUp(ev)
			ev.nextAction = SendNack
			break
		}
	}

	r.startTimer()
}

func newRecovery(ttl int, cxt *Context) *recovery {

	domain := newDomain(ttl)
	r := recovery{cxt: cxt, ttl: ttl, domain: domain}

	/* the loss table is shared */

	domain.lossTab = &lossTable{}
	domain.lossHistory = &lossHistory{}

	if ttl > 63 {
		domain.child = newDomain(63)
		domain.child.lossTab = domain.lossTab
		domain.child.lossHistory = domain.lossHistory
		domain.setChild(domain.child)
		domain = domain.child
	}
	if ttl > 47 {
		domain.child = newDomain(47)
		domain.child.lossTab = domain.lossTab
		domain.child.lossHistory = domain.lossHistory
		domain.setChild(domain.child)
		domain = domain.child
	}
	if ttl > 15 {
		domain.child = newDomain(15)
		domain.child.lossTab = domain.lossTab
		domain.child.lossHistory = domain.lossHistory
		domain.setChild(domain.child)
		domain = domain.child
	}

	return &r
}

func (r *recovery) stop() {
	if r.task != nil {
		timer.recallTimer(r.task)
		r.task = nil
		r.domain.lossTab.clear()
	}

}
func (r *recovery) processNack(received *lossEvent) {
	dc := r.lookupDomain(received.scope)

	received.domain = dc
	dc.stats.nack++

	/*
	 * there are three cases:
	 * 1. we'r a receiver having the same loss, cancel the next NACK
	 * and schedule another timer.
	 * 2. we'r the sender, send repairs immediately.
	 * 3. we'r a receiver without the same loss, schedule the
	 * sending of repairs.
	 */
	event := r.lookup(received.source, r.cxt.whoami)

	if event != nil {

		/*
		 * if the received NACK contains the local NACK, cancel the current
		 * timer and delay next NACK if the local id is greater.
		 */
		if received.contains(event) {
			if event.nextAction == SendNack {
				r.goUp(event)

				rcv := received.reporter.getID() & 0xffffffff
				me := r.cxt.whoami.getID() & 0xffffffff

				if me > rcv {
					event.nextAction = DelayAndStay
				}
			}
		}
		if event.contains(received) {
			slice := dc.stats.mrtt >> 2

			if received.source.interval < 200 {
				slice += received.source.interval
			} else {
				slice += 200
			}
			if int(millis(received.rcvSendTime.Sub(event.rcvSendTime))) <= slice {
				dc.stats.dupNack++

				if isDebug() {
					logDebug("Dup nack: ", slice, "/", dc.stats.mrtt)
				}
			}

			return
		}
	}

	/* ignore duplicates */

	if dc.isDuplicate(received) {
		dc.stats.dupNack++

		return
	}

	/*
	 * if I'm the sender, send repair immediately.
	 * otherwise schedule resend.
	 */
	if r.cxt.whoami == received.source {

		// if (received.domain.parent == null)			/* test XXXXXXXXX */

		r.resend(received)
	} else {

		/*
		 * at the top level, don't send repair if we are not the original sender.
		 * check cache now, since we may send recently received repairs.
		 */
		if dc.parent != nil && r.cxt.profile.sendRepair {

			/* make a copy to keep the original intact (kept in history) */

			received := &(*received)

			var firstSent int64
			var bitsSent uint32

			if received.source.isCached(received.low) {
				firstSent = received.low
			}

			for i := 0; i < 32; i++ {
				if ((received.bitmask >> uint(i)) & 0x01) > 0 {
					seqno := received.low + int64(i) + 1

					if received.source.isCached(seqno) {
						if firstSent == 0 {
							firstSent = seqno
						} else {
							bitsSent |= 0x1 << uint(seqno-firstSent-1)
						}
					}
				}
			}

			if firstSent > 0 {
				received.low = firstSent
				received.bitmask = bitsSent

				dc.lossTab.add(received)
				r.resendTimer(received)
				r.startTimer()
			}
		}
	}
}

func (r *recovery) processNackReply(entity Entity, event *lossEvent, delay int) {
}

/**
 * handles a local loss event detected when receiving data from the
 * given source. Keep only one loss event in the queue for a sender.
 */
func (r *recovery) handleLoss(s *sender) {
	if r.cxt.profile != nil && r.cxt.profile.lossAllowed() {
		return
	}

	diff := diff32(s.maxseq, s.expected)

	if diff > s.cacheSize {

		/*
		 * lost too many, can't repair.
		 */
		r.cxt.lrmp.handleSyncError(s, BufferOverrun)

		return
	}

	/*
	 * find if there is already one for the same source.
	 */
	ev := r.lookup(s, r.cxt.whoami)

	if ev == nil {
		d := r.getDomain()

		ev = newLossEvent(s)
		ev.reporter = r.cxt.whoami
		ev.scope = d.scope
		ev.domain = d

		ev.computeBitmask()
		d.lossTab.add(ev)

		if isDebug() {
			logDebug("new loss ", ev)
		}

		/* schedule a timer */

		r.nackTimer(ev)
		r.startTimer()
	} else {

		/*
		 * new data is still arriving, that means the sender is active,
		 * decrement the timer.
		 */
		if ev.nackCount > 0 {
			ev.nackCount--
		}

		ev.nextAction = SendNack
	}
}

/**
* lookups a domain which matches the given scope, provided that the lookup
* begins from the lowest domain.
 */
func (r *recovery) lookupDomain(ttl int) *domain {
	for d := r.domain; d != nil; d = d.parent {
		if d.parent == nil || ttl <= d.scope {
			return d
		}
	}

	return nil
}

func (r *recovery) lookup(s *sender, reporter Entity) *lossEvent {
	return r.domain.lossTab.lookup(s, reporter)
}

/**
* returns the lowest enabled domain.
 */
func (r *recovery) getDomain() *domain {
	for d := r.domain; d != nil; d = d.parent {
		d.checkState()

		if d.stats.enabled {
			return d
		}
	}

	return nil
}

/*
 * determine the timer value for sending NACK. This is an exponential back-off
 * timer, each time the timeout elapsed, the timeout value T(i) is set to T(i-1)*2
 * until it reaches the upper bound. Three parameters to be taken into account:
 * - mrtt,
 * - nack count,
 * - transmission interval used by the source.
 * For the first nack, we can use the mrtt to quickly report the loss.
 * The tricky part is sending the following nacks. The timer value must be larger
 * than the resend timer and the time in which the source may send repairs. So
 * a transmission interval must be added. The initial mrtt is used as the lower
 * bound since responders may use it to schedule the resend.
 */
func (r *recovery) nackTimer(ev *lossEvent) {
	d := (ev.domain.stats.mrtt << ev.nackCount) >> 3

	/*
	 * at the moment we are rather conservative, but at some later time
	 * the low bound may be removed after tests for better performance.
	 */
	if ev.nackCount > 0 || (ev.domain.child != nil && ev.domain.child.isEnabled()) {
		if d < ev.domain.initialMRTT {
			d = ev.domain.initialMRTT
		}

		d = int(float64(d) * (1.0 + rand.Float64()))

		if ev.source.interval < 200 {
			d += ev.source.interval
		} else {
			d += 200
		}
	} else {
		d = int(float64(d) * (1.0 + rand.Float64()))
	}
	if isDebug() {
		logDebug("NACK timer=", d, " #", ev.nackCount, " ", ev.domain.stats.getRTT(), "/", ev.source.interval, "@", ev.scope)
	}

	ev.timeoutTime = time.Now().Add(time.Duration(d) * time.Millisecond)
}

func (r *recovery) startTimer() {
	future := time.Time{}

	var event *lossEvent

	for elem := r.domain.lossTab.Front(); elem != nil; elem = elem.Next() {
		event = elem.Value.(*lossEvent)
		if future.IsZero() || event.timeoutTime.Before(future) {
			future = event.timeoutTime
		}
	}

	if !future.IsZero() {
		if r.task != nil {
			timer.recallTimer(r.task)
		}

		millis := millis(future.Sub(time.Now()))

		if millis < 0 {
			millis = 1
		}

		r.task = timer.registerTimer(int(millis), r, nil)

		if isDebug() {
			logDebug("Next timeout=", millis, " events: ", r.domain.lossTab.Len())
		}
	}
}

func (r *recovery) heardRepair(p *Packet, dup bool) {
	dc := r.lookupDomain(p.scope)

	dc.stats.repairPackets++
	dc.stats.repairBytes += int64(p.datalen)

	if p.sender != p.source {
		dc.stats.thirdPartyRepairs++
	}

	source := p.source.(*sender)

	if dup {
		dc.stats.dupPackets++
		dc.stats.dupBytes += int64(p.datalen)

		if p.sender != p.source {
			dc.stats.thirdPartyDuplicates++
		}
		if isDebug() {
			if p.sender == source {
				logDebug("duplicate #", p.seqno, " from source")
			} else {
				logDebug("duplicate #", p.seqno, " from third party")
			}
		}

		/* unconditionally cancel resend the same seqno */

		r.cxt.sender.cancelResend(source, p.seqno, p.scope)

		/* unconditionally cancel resend if not started reply XXXX */

		/* conditionall cancel if local id is higher */

		if source != r.cxt.whoami {
			p1 := source.getPacket(p.seqno)

			if p1 != nil {
				rcv := p.sender.getID() & 0xffffffff
				me := r.cxt.whoami.getID() & 0xffffffff

				if me > rcv || p.sender == source {
					r.cxt.sender.cancelResendByID(source, p1.retransmitID, p.scope)
				}
			}
		}
	} else {
		event := r.lookup(source, r.cxt.whoami)

		/*
		 * as the responder is sending repairs, delay the next NACK since
		 * the following repairs may be in the send queue of the responder.
		 */
		if event != nil {
			if event.high <= event.source.expected {

				/* the loss has been repaired */

				r.domain.lossTab.remove(event)
			} else if (p.seqno - event.low) < 33 {
				event.nextAction = DelayAndGoDown
			} else {
				event.nextAction = DelayAndStay
			}
		}
		if !dc.stats.enabled {
			dc.enable()
		} else {
			dc.failedNack = 0
		}
	}

}

/*
 * allow partial resend.
 */
func (r *recovery) resend(ev *lossEvent) {
	var firstSent int64
	var bitsSent uint32

	if r.resendBySeqno(ev.low, ev) {
		firstSent = ev.low
	}

	for i := 0; i < 32; i++ {
		if ((ev.bitmask >> uint(i)) & 0x01) > 0 {
			seqno := ev.low + int64(i) + 1

			if r.resendBySeqno(seqno, ev) {
				if firstSent == 0 {
					firstSent = seqno
				} else {
					bitsSent |= 0x1 << uint(seqno-firstSent-1)
				}
			}
		}
	}

	if isDebug() {
		logDebug("send R_NACK ", ev.reporter)
	}

	/* send NACK reply if did resend */

	if firstSent > 0 {
		reply := NewPacket(false, 64)

		reply.scope = ev.scope
		reply.offset = 0

		reply.appendNackReply(ev, r.cxt.whoami, int(firstSent), bitsSent)
		r.cxt.lrmp.sendControlPacket(reply, ev.scope)

		ev.domain.stats.nackReply++
	}
}

/*
 * really resend repairs if
 * o the packet is cached and
 * o last send is in a scope smaller than reported scope or
 * o no repair received after the reception time of the event.
 * return true if the packet is resent.
 */
func (r *recovery) resendBySeqno(seqno int64, ev *lossEvent) bool {
	p := ev.source.getPacket(seqno)

	if p == nil {
		if isDebug() {
			logDebug("unable resend #", seqno)
		}

		return false
	}

	p.retransmitID = int(ev.low)

	r.cxt.sender.enqueueResend(p, ev.scope)

	return true
}

func (r *recovery) resendTimer(ev *lossEvent) {
	d := ev.domain.stats.mrtt >> 3

	d = int(float64(d) * (1.0 + rand.Float64()))

	/*
	 * add a transmission interval since the sender may resend.
	 */
	if ev.source.interval < 200 {
		d += ev.source.interval
	} else {
		d += 200
	}
	if isDebug() {
		logDebug("resendTimer=", d, " ", ev.domain.stats.mrtt, "/", ev.source.interval)
	}

	ev.timeoutTime = addMillis(time.Now(), d)
}
func (r *recovery) goUp(ev *lossEvent) {
	if ev.domain.parent != nil && ev.scope < ev.source.distance {
		ev.scope = ev.domain.parent.scope
		ev.domain = ev.domain.parent
		ev.nackCount = 0
	} else {
		ev.nackCount++
	}

	r.nackTimer(ev)
}
func (r *recovery) goDown(ev *lossEvent) {
	if ev.domain.child != nil && ev.domain.child.isEnabled() {
		ev.scope = ev.domain.child.scope
		ev.domain = ev.domain.child
		ev.nackCount = 1
	} else {
		ev.nackCount++
	}

	r.nackTimer(ev)
}
