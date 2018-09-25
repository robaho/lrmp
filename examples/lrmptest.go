package main

import (
	"github.com/robaho/lrmp"
	"log"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}

	profile := lrmp.Profile{}

	lrmp, err := lrmp.NewLrmp("225.0.0.100", 6000, 0, "en0", profile)
	if err != nil {
		log.Fatal(err)
	}
	lrmp.Start()

	wg.Add(1)
	wg.Wait()

}
