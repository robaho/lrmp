package lrmp

import "errors"

var Version = "LRMP-1.4.2"

type Lrmp struct {
	impl *impl
}

// create and join an LRMP session
func NewLrmp(addr string, port int, ttl int, network string, profile Profile) (*Lrmp, error) {
	impl, err := newImpl(addr, port, ttl, network, profile)
	if err != nil {
		return nil, err
	}

	lrmp := Lrmp{impl}
	return &lrmp, nil
}

func (l *Lrmp) Start() {
	l.impl.startSession()
}
func (l *Lrmp) Stop() {
	l.impl.stopSession()
}

func (l *Lrmp) Stats() Stats {
	return l.impl.stats
}
func (l *Lrmp) DomainStats(scope int) DomainStats {
	return l.impl.domainStats
}
func (l *Lrmp) WhoAmI() Entity {
	return l.impl.whoAmI()
}

func (l *Lrmp) Send(packet *Packet) error {
	if packet.getDataLength() > packet.getMaxDataLength() {
		return errors.New("bad packet length")
	}
	return l.impl.send(packet)
}
func (l *Lrmp) Flush() {
	l.impl.flush()
}
