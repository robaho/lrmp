package main

import (
	"bufio"
	"fmt"
	"github.com/robaho/lrmp"
	"log"
	"os"
)

type handler struct {
}

func (handler) ProcessData(p *lrmp.Packet) {
	fmt.Println("got a packet", string(p.GetDataBuffer()))
}

func (handler) ProcessEvent(event int, data interface{}) {
	fmt.Println("got an event", event, data)
}

func main() {
	fmt.Println("type a message and press 'enter' to send")

	profile := lrmp.NewProfile()
	profile.Handler = new(handler)

	l, err := lrmp.NewLrmp("225.0.0.100", 6000, 0, "en0", *profile)
	if err != nil {
		log.Fatal(err)
	}
	l.Start()

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		s := scanner.Text()

		bytes := []byte(s)

		p := lrmp.NewPacket(true, len(bytes))
		copy(p.GetDataBuffer(), bytes)
		p.SetDataLength(len(bytes))

		fmt.Println("sending message 100 times")

		for i := 0; i < 100; i++ {
			l.Send(p)
		}
	}

	l.Stop()

}
