package lrmp

import "time"

type Stats struct {
	badLength              int
	badVersion             int
	ctrlPackets            int
	ctrlBytes              int
	dataPackets            int
	dataBytes              int
	senderReports          int
	rrSelect               int
	receiverReports        int
	populationEstimate     int
	populationEstimateTime time.Time
	failures               int
}
type DomainStats struct {
	childScope int
	enabled    bool
	mrtt       int // in 1/8 miilisecs
}

func (stats *DomainStats) getRTT() int {
	return stats.mrtt >> 3
}
