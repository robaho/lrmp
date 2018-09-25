package lrmp

type Context struct {
	whoami  *sender
	profile Profile
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
var timer = newEventManager()

const (
	BigDecrease    = 2
	MediumDecrease = 4
	SmallDecrease  = 6
	None           = 8
	SmallIncrease  = 9
	MediumIncrease = 12
	BigIncrease    = 16
)

func newContext() *Context {
	ctx := Context{}
	ctx.senderReportInterval = 4000
	ctx.rcvReportSelInterval = 30000
	ctx.sndInterval = 100
	ctx.adjust = SmallIncrease
	return &ctx
}
