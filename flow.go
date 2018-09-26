package lrmp

type flow struct {
	cxt *Context
}

func newFlow(cxt *Context) *flow {
	f := flow{cxt: cxt}

	go func() {
		for {
			p := <-f.cxt.sendQueue
			f.cxt.lrmp.session.send(p.buff, p.len, p.scope)
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
