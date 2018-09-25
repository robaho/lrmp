package lrmp

import "math/rand"

type recovery struct {
	cxt    *Context
	ttl    int
	domain *domain
	random rand.Rand
	event  *lossEvent
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
	if r.event != nil {
		r.cxt.timer.recallTimer(r.event)
		r.event = nil
		r.domain.lossTab.clear()
	}

}
func (r *recovery) processNack(event *lossEvent) {
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

		nackTimer(ev)
		startTimer()
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
func (r *recovery) lookupDomain(scope int) *domain {
}
func (r *recovery) lookup(s *sender, whoami *sender) *lossEvent {
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

		d = int(d * (1.0 + rand.Float64()))

		if ev.source.interval < 200 {
			d += ev.source.interval
		} else {
			d += 200
		}
	} else {
		d = (int)(d * (1.0 + rand.Float64()))
	}
	if isDebug() {
		logDebug("NACK timer=", d, " #", ev.nackCount, " ", ev.domain.stats.getRTT(), "/", ev.source.interval, "@", ev.scope)
	}

	ev.timeoutTime = System.currentTimeMillis() + d
}
