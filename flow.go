package lrmp

import "time"

type flow struct {
	cxt         *Context
	lastPackets int
	lastBytes   int
	lastTime    time.Time
}

func newFlow(cxt *Context) *flow {
	f := flow{cxt: cxt}

	go func() {
		for {
			pack := <-f.cxt.sendQueue

			f.resend() // always check resend

			if pack == nil { // might be wakeup from resend queue
				continue
			}

			/* send a packet */

			pack.source = cxt.whoami
			pack.sender = pack.source

			if pack.reliable {
				pack.seqno = f.cxt.whoami.expected

				cxt.whoami.incExpected()

				/*
				 * as we know the sequence number is incremented by one, we can
				 * safely append the packet to the send window which keeps a pool
				 * of packets sorted by seqno.
				 */
				cxt.whoami.appendPacket(pack)

				if isDebug() {
					logDebug("sending #", pack.seqno, " len=", pack.GetDataLength())
				}
			}

			pack.scope = cxt.lrmp.ttl

			cxt.lrmp.sendDataPacket(pack, false)

			f.flowControl()
			f.throttle()
		}
	}()

	return &f
}

func (f *flow) enqueue(p *Packet) {
	f.cxt.sendQueue <- p
}
func (f *flow) flush() {
}

func (f *flow) stop() {
	close(f.cxt.sendQueue)
}

func (f *flow) throttle() {
	if f.cxt.profile.Throughput != BestEffort && f.cxt.sndInterval > 0 {
		time.Sleep(time.Duration(f.cxt.sndInterval) * time.Millisecond)
	}
}

func (f *flow) resend() {
	for {
		pack := f.cxt.resendQueue.dequeue()

		if pack == nil {
			break
		}
		if isDebug() {
			logDebug("resending #", pack.seqno, " @", pack.scope)
		}

		pack.sender = f.cxt.whoami

		f.cxt.lrmp.sendDataPacket(pack, true)

		if !f.cxt.resendQueue.isEmpty() {
			f.flowControl()
			f.throttle()
		} else {
			break
		}
	}
}

func (f *flow) flowControl() {
	cxt := f.cxt

	pcount := cxt.whoami.packets - f.lastPackets

	if pcount < cxt.checkInterval {
		return
	}

	f.lastPackets = cxt.whoami.packets

	bcount := cxt.whoami.bytes - f.lastBytes

	f.lastBytes = cxt.whoami.bytes

	cur := time.Now()

	cxt.actualRate = bcount * 1000 / int(millis(cur.Sub(f.lastTime)))

	cxt.whoami.setRate(cxt.actualRate)

	f.lastTime = cur

	if cxt.profile.Throughput == ConstantThroughput {
		return
	}

	cxt.curRate = (cxt.curRate * cxt.adjust) >> 3

	if cxt.curRate < cxt.profile.minRate {
		cxt.curRate = cxt.profile.minRate
	} else if cxt.curRate > cxt.profile.maxRate {
		cxt.curRate = cxt.profile.maxRate
	}

	cxt.adjust = SmallIncrease

	if cxt.whoami.bytes > 0 {
		cxt.sndInterval = (bcount * 1000 / pcount) / cxt.curRate
	}

	/* due to CPU load and break */

	if cxt.actualRate < ((cxt.curRate * 3) / 4) {
		cxt.sndInterval = (cxt.sndInterval * 3) / 4
	}
	if cxt.sndInterval > 30000 {
		cxt.sndInterval = 30000
	}
	if isDebug() {
		logDebug("rate/interval: ", cxt.curRate, "/", cxt.sndInterval)
	}
}
func (f *flow) enqueueResend(pack *Packet, scope int) {
	if f.cxt.resendQueue.contains(pack) {
		if pack.scope < scope {
			pack.scope = scope
		}
		return
	}

	if isDebug() {
		logDebug("enqueue resend seq# ", pack.seqno)
	}

	pack.scope = scope

	f.cxt.resendQueue.enqueue(pack)

	f.cxt.sendQueue <- nil
}

func (f *flow) cancelResend(s *sender, seqno int64, scope int) {
	if isDebug() {
		logDebug("cancel resend seq# ", seqno)
	}
	f.cxt.resendQueue.remove(s, seqno, scope)
}

func (f *flow) cancelResendByID(s *sender, id int, scope int) {
	if isDebug() {
		logDebug("cancel resend retransmit ID# ", id)
	}
	f.cxt.resendQueue.cancel(s, id, scope)
}
