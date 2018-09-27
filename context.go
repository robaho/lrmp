package lrmp

import "net"

type Context struct {
	whoami  *sender
	profile *Profile
	stats   Stats

	/* control objects */

	lrmp    *impl
	sender  *flow
	recover *recovery
	sm      *entityManager

	/* flow/congestion control data, rate is in bytes/sec */

	adjust        int /* scaled by a factor of 8 */
	curRate       int
	actualRate    int
	checkInterval int
	sndInterval   int

	/* output */

	sendQueue            chan *Packet
	resendQueue          packetQueue
	senderReportInterval int
	rcvReportSelInterval int
}

var maxQueueSize = 16
var timer = newTimerManager()

const (
	BigDecrease    = 2
	MediumDecrease = 4
	SmallDecrease  = 6
	None           = 8
	SmallIncrease  = 9
	MediumIncrease = 12
	BigIncrease    = 16
)

func newContext(ip net.IP, ttl int) *Context {
	ctx := Context{}
	ctx.sendQueue = make(chan *Packet, 1000)
	ctx.senderReportInterval = 4000
	ctx.rcvReportSelInterval = 30000
	ctx.sndInterval = 100
	ctx.adjust = SmallIncrease
	ctx.sm = newEntityManager(ip)
	ctx.sender = newFlow(&ctx)
	ctx.recover = newRecovery(ttl, &ctx)
	return &ctx
}

func (c *Context) setProfile(prof *Profile) {

	/* keep a cloned profile to prevent change by upper layer */

	profile := *prof

	if profile.sendWindowSize >= 32 {
		c.whoami.cacheSize = profile.sendWindowSize
		c.whoami.initCache(profile.sendWindowSize)
	}

	c.sm.profile = &profile
	c.profile = c.sm.profile

	if isDebug() {
		logDebug("rcv/snd window:", profile.rcvWindowSize, "/", profile.sendWindowSize)
	}

	/*
	 * check the data rate and converts kilo bits/sec to bytes/sec.
	 */
	if profile.minRate <= 0 {

		/* the rate should be greater than zero */

		profile.minRate = 125
	} else {
		profile.minRate = (profile.minRate * 1000) / 8
	}

	profile.maxRate = (profile.maxRate * 1000) / 8

	if profile.maxRate <= profile.minRate {
		profile.maxRate = profile.minRate
	}

	/* init for the first time only */

	if c.curRate == 0 {
		c.curRate = (profile.minRate + profile.maxRate) / 2

		if c.curRate < profile.minRate {
			c.curRate = profile.minRate
		}

		c.sndInterval = MTU * 1000 / c.curRate
	}

	c.checkInterval = profile.sendWindowSize / 8

	if c.checkInterval < 4 {
		c.checkInterval = 4
	}
	if isDebug() {
		logDebug("min/cur/max rate: ", profile.minRate, "/", c.curRate, "/", profile.maxRate, " send/check interval: ", c.sndInterval, "/", c.checkInterval)
	}
}
