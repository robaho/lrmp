package lrmp

import (
	"fmt"
	"strconv"
	"time"
)

const (
	SendNack       = 0
	DelayAndStay   = 1
	DelayAndGoUp   = 2
	DelayAndGoDown = 3
)

type lossEvent struct {
	source      *sender
	rcvSendTime time.Time
	low         int64
	high        int64
	bitmask     uint32
	scope       int
	reporter    Entity
	timestamp   int
	nextAction  int
	nackCount   uint
	domain      *domain
	timeoutTime time.Time
}

func (ev *lossEvent) String() string {
	return fmt.Sprint(ev.reporter, " -> ", ev.source, " : ", ev.low, "/", strconv.FormatUint(uint64(ev.bitmask), 16), "@", ev.scope)
}

func newLossEvent(e *sender) *lossEvent {
	le := lossEvent{source: e}
	return &le
}

func (ev *lossEvent) computeBitmask() {
	ev.low = ev.source.expected

	maxdiff := diff32(ev.source.maxseq, ev.low)

	if maxdiff < 0 {
		ev.low = -1 /* no loss */

		return
	} else if maxdiff > 32 {
		maxdiff = 32
	}

	/*
	 * set a bit to 1 for a packet lost.
	 */
	ev.high = ev.low
	ev.bitmask = 0

	for i := 1; i <= maxdiff; i++ {
		if !ev.source.isCached(ev.low + int64(i)) {
			ev.bitmask |= uint32(0x1 << uint(i-1))
			ev.high = ev.low + int64(i)
		}
	}
}
func (event *lossEvent) contains(ev *lossEvent) bool {
	diff := uint32(diff32(ev.low, event.low))

	if diff == 0 {
		return (ev.bitmask & ^event.bitmask) == 0
	} else if diff > 0 {
		diff = event.bitmask >> (diff - 1)

		if (diff & 0x01) > 0 {
			diff >>= 1

			return (ev.bitmask & ^diff) == 0
		}
	}

	return false
}
