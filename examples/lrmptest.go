package main

import (
	"bufio"
	"fmt"
	"github.com/robaho/lrmp"
	"log"
	"os"
	"strconv"
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

		fmt.Println("sending message ", s, " 100 times")

		for i := 0; i < 100; i++ {
			bytes := []byte(s + " #" + strconv.Itoa(i))

			p := lrmp.NewPacket(true, len(bytes))
			copy(p.GetDataBuffer(), bytes)
			p.SetDataLength(len(bytes))
			l.Send(p)
		}
	}

	l.Stop()

}
