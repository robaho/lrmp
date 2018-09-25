package lrmp

const (
	LossAllowed            = 1
	LimitedLoss            = 2
	NoLoss                 = 3
	BestEffort             = 1
	ConstantThroughput     = 2
	AdaptedThroughput      = 3
	NoReceiverReport       = 1
	RandomReceiverReport   = 2
	PeriodicReceiverReport = 3
)

type Profile struct {
	Handler        EventHandler
	sendWindowSize int
	rcvWindowSize  int
	minRate        int
	maxRate        int
	sendRepair     bool
	Ordered        bool
	Reliability    int
	Throughput     int
	handler        EventHandler
}

func (profile *Profile) lossAllowed() bool {

}

func NewProfile() *Profile {
	p := Profile{sendWindowSize: 64, rcvWindowSize: 64, minRate: 8, maxRate: 64, sendRepair: true, Ordered: true, Reliability: NoLoss, Throughput: AdaptedThroughput}
	return &p
}

type EventHandler interface {
	ProcessData(p *Packet)
	ProcessEvent(event int, data interface{})
}
