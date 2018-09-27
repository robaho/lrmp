package lrmp

import (
	"container/list"
	"time"
)

const (
	MaxDisableTime = 180000 /* millis */
	DisableTries   = 5
	MinRTTValue    = 2
	MaxRTTValue    = 16000
)

type lossHistory struct {
	list.List
}

func (lh *lossHistory) contains(ev *lossEvent) bool {
	for next := lh.Front(); next != nil; next = next.Next() {
		if next.Value == ev {
			return true
		}
	}
	return false
}

const historySize = 16

type domain struct {
	lastTimeToggle time.Time
	failedNack     int
	lossTab        *lossTable
	child          *domain
	lossHistory    *lossHistory
	parent         *domain
	stats          DomainStats
	scope          int
	initialMRTT    int
}

func (d *domain) updateMRTT(rtt int) {
}
func (d *domain) setChild(child *domain) {
	d.child = child
	d.child.parent = d
	d.stats.childScope = child.scope

}
func (d *domain) checkState() {

	/* top level domain always enabled */

	if d.parent == nil {
		return
	}
	if d.stats.enabled {
		if d.failedNack > DisableTries {
			d.disable()
		}
	} else {
		if time.Now().Sub(d.lastTimeToggle) > time.Duration(time.Millisecond*MaxDisableTime) {
			d.enable()
		}
	}
}

func (d *domain) enable() {
	if d.stats.enabled {
		return
	}

	d.stats.enabled = true
	d.failedNack = 0
	d.lastTimeToggle = time.Now()

	if d.parent != nil {
		d.parent.enable()
	}
	if isDebug() {
		logDebug("ENABLE scope=", d.scope)
	}
}
func (d *domain) disable() {
	if d.parent == nil || !d.stats.enabled {
		return
	}

	d.stats.enabled = false
	d.lastTimeToggle = time.Now()

	if d.child != nil {
		d.child.disable()
	}
	if isDebug() {
		logDebug("DISABLE ", d.scope, " fails=", d.failedNack)
	}
}
func (d *domain) isEnabled() bool {
	return d.stats.enabled

}
func (d *domain) isDuplicate(event *lossEvent) bool {
	dup := false
	slice := d.stats.mrtt >> 3

	if event.source.interval < 200 {
		slice += event.source.interval
	} else {
		slice += 200
	}

	for elem := d.lossHistory.Back(); elem != nil; elem = elem.Prev() {
		ev1 := elem.Value.(*lossEvent)

		if ev1.source != event.source {
			continue
		}

		diff := int(millis(event.rcvSendTime.Sub(ev1.rcvSendTime)))

		if diff < slice {
			if ev1.contains(event) {
				dup = true

				if isDebug() {
					logDebug("Dup NACK: ", diff, "<", slice)
				}
				break
			}
		} else {

			/* keep the most recent */

			if event.contains(ev1) {
				if isDebug() {
					logDebug("Repeated nack in ", diff)
				}

				d.lossHistory.Remove(elem)
			}
		}
	}

	if d.lossHistory.Len() > historySize {
		d.lossHistory.Remove(d.lossHistory.Front())
	}

	d.lossHistory.PushBack(event)

	return dup
}

func newDomain(ttl int) *domain {
	d := domain{scope: ttl}

	d.stats.enabled = true
	d.stats.childScope = 0

	/* 200*(scope/63)^2 */

	d.initialMRTT = getInitialRTT(d.scope)
	d.stats.mrtt = d.initialMRTT << 3

	return &d
}

func getInitialRTT(ttl int) int {
	if ttl <= 15 {
		return 12
	} else if ttl >= 126 {
		return 800
	}

	return (200*ttl*ttl + 1984) / 3969
}
