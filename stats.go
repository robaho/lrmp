package lrmp

import "time"

type Stats struct {
	badLength              int
	badVersion             int
	ctrlPackets            int
	ctrlBytes              int64
	dataPackets            int
	dataBytes              int64
	senderReports          int
	rrSelect               int
	receiverReports        int
	populationEstimate     int
	populationEstimateTime time.Time
	failures               int
	outOfBand              int
}
type DomainStats struct {
	childScope    int
	enabled       bool
	mrtt          int // in 1/8 miilisecs
	repairPackets int
	repairBytes   int64
}

func (stats *DomainStats) getRTT() int {
	return stats.mrtt >> 3
}
