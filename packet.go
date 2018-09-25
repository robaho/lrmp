package lrmp

import "time"

type Packet struct {
	len          int
	isReliable   bool
	seqno        int64
	scope        int
	retransmitID int
	offset       int
	buff         []byte
	maxDataLen   int
	datalen      int
	source       Entity
	sender       Entity
	rcvSendTime  time.Time
	retransmit   bool
}

func (packet *Packet) getDataLength() int {
	return packet.len
}
func (packet *Packet) getMaxDataLength() int {
	return packet.maxDataLen
}

const MTU = 1400

func newPacket(reliable bool, length int) *Packet {

	p := Packet{}

	p.isReliable = reliable

	if reliable {
		p.offset = 16
	} else {
		p.offset = 8
	}

	size := p.offset + length

	/* mod 4 */

	size = (size + 3) & 0xfffc

	if size > MTU {
		size = MTU
	}

	p.buff = make([]byte, size)
	p.maxDataLen = size - p.offset
	p.datalen = 0

	return &p
}

func newDataPacket(reliable bool, buff []byte, offset int, len int) *Packet {
	p := Packet{}

	p.buff = buff
	p.isReliable = reliable

	if reliable {
		p.datalen = len - 16
		p.offset = offset + 16
	} else {
		p.datalen = len - 8
		p.offset = offset + 8
	}

	/* padding */

	if (buff[offset] & 0x20) > 0 {
		p.datalen -= int(buff[offset+len-1] & 0xff)
	}

	p.scope = int(buff[offset+1] & 0xff)
	p.rcvSendTime = time.Now()

	return &p
}
