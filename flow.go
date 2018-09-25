package lrmp

type flow struct {
	cxt Context
}

func (f *flow) enqueue(p *Packet) {
	f.cxt.sendQueue <- p

	go start()
}
func (f *flow) flush() {
}

func start() {
}
