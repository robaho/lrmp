package lrmp

import "time"

const (
	MaxDisableTime = 180000 /* millis */
	DisableTries   = 5
	MinRTTValue    = 2
	MaxRTTValue    = 16000
)

type lossHistory []*lossEvent

const historySize = 16

type domain struct {
	ttl            int
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

func newDomain(ttl int) *domain {
	d := domain{ttl: ttl}
	return &d
}
