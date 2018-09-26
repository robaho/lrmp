package main

import (
	"fmt"
	"github.com/robaho/lrmp"
	"log"
	"sync"
)

type handler struct {
}

func (handler) ProcessData(p *lrmp.Packet) {
	fmt.Println("got a packet", string(p.GetData()))
}

func (handler) ProcessEvent(event int, data interface{}) {
	fmt.Println("got an event", event, data)
}

func main() {
	wg := sync.WaitGroup{}

	profile := lrmp.NewProfile()
	profile.Handler = new(handler)

	lrmp, err := lrmp.NewLrmp("225.0.0.100", 6000, 0, "en0", *profile)
	if err != nil {
		log.Fatal(err)
	}
	lrmp.Start()

	wg.Add(1)
	wg.Wait()

}
