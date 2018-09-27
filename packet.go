package lrmp

import (
	"errors"
	"strconv"
	"time"
)

type Packet struct {
	reliable     bool
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

const padBit = 0x20

func (packet *Packet) GetDataLength() int {
	return packet.datalen
}
func (packet *Packet) GetMaxDataLength() int {
	return packet.maxDataLen
}

func (packet *Packet) GetDataBuffer() []byte {
	return packet.buff[packet.offset : packet.offset+packet.maxDataLen]
}

func (packet *Packet) SetDataLength(length int) error {
	if length > packet.maxDataLen {
		return errors.New("maximum data length of " + strconv.Itoa(packet.maxDataLen))
	}
	packet.datalen = length
	return nil
}

func (p *Packet) appendSenderReport(whoami *sender) {

	offset := p.offset
	buff := p.buff

	start := offset

	buff[offset] = byte((VersionNumber << 6) | SR_PT)
	offset++
	buff[offset] = byte(p.scope)
	offset += 3

	intToByte(int(whoami.getID()), buff, offset)

	offset += 4

	intToByte(ntp32(nowMillis()), buff, offset)

	offset += 4

	intToByte(int(whoami.expected), buff, offset)

	offset += 4

	intToByte(whoami.packets, buff, offset)

	offset += 4

	intToByte(whoami.bytes, buff, offset)

	offset += 4

	/* fill the length field */

	len := offset - start

	shortToByte(len, buff, start+2)

	p.offset = offset
}

func (p *Packet) appendRRSelection(whoami *sender, prob int, period int) {
	offset := p.offset
	buff := p.buff

	start := offset

	buff[offset] = (byte)((VersionNumber << 6) | RS_PT)
	offset++
	buff[offset] = byte(p.scope)
	offset += 3

	intToByte(int(whoami.getID()), buff, offset)

	offset += 4

	intToByte(ntp32(nowMillis()), buff, offset)

	offset += 4

	/* probability */

	shortToByte(prob, buff, offset)
	offset += 2

	/* period */

	shortToByte(period, buff, offset)
	offset += 2

	buff[offset] = byte(0xff)
	offset++
	buff[offset] = byte(0xff)
	offset++
	buff[offset] = byte(0xff)
	offset++
	buff[offset] = byte(0xff)
	offset++

	/* fill the length field */

	len := offset - start

	shortToByte(len, buff, start+2)

	p.offset = offset
}

func (p *Packet) appendReceiverReport(sender *sender, whoami *sender) {
	start := p.offset

	offset := p.offset
	buff := p.buff

	buff[offset] = (byte)((VersionNumber << 6) | RR_PT)
	offset++
	buff[offset] = byte(p.scope)
	offset += 3

	intToByte(int(whoami.getID()), buff, offset)

	offset += 4

	intToByte(int(sender.getID()), buff, offset)

	offset += 4

	intToByte(sender.rrTimestamp, buff, offset)

	offset += 4

	delay := time.Now().Sub(sender.rrSelectTime)

	delayMS := millisToFixedPoint32(int(millis(delay)))

	intToByte(delayMS, buff, offset)

	offset += 4

	intToByte(int(sender.expected), buff, offset)

	offset += 4

	absLost := int(sender.maxseq-sender.startseq) + 1 - (sender.packets - sender.duplicates)
	relativeLost := absLost - sender.rrAbsLost

	sender.rrAbsLost = absLost

	if relativeLost > 0 {
		expected := int(sender.maxseq - sender.rrMaxSeqno)

		sender.rrMaxSeqno = sender.maxseq

		if expected > relativeLost {
			buff[offset] = byte((relativeLost << 8) / expected)
			offset++
		} else {
			buff[offset] = byte(0xff)
			offset++
		}
	} else {
		buff[offset] = 0
		offset++
	}

	if isDebug() {
		logDebug("send RR lost/rate:", absLost, "/",
			float64(buff[offset-1])/256.0, " max/init:",
			sender.maxseq, "/", sender.startseq,
			" packs/dup:", sender.packets, "/",
			sender.duplicates)
	}
	if absLost > 0 {
		buff[offset] = byte((absLost >> 16) & 0xff)
		offset++
		buff[offset] = byte((absLost >> 8) & 0xff)
		offset++
		buff[offset] = byte(absLost & 0xff)
		offset++
	} else {
		buff[offset] = 0
		offset++
		buff[offset] = 0
		offset++
		buff[offset] = 0
		offset++
	}

	/* fill the length field */

	len := offset - start

	shortToByte(len, buff, start+2)

	p.offset = offset

}

const MTU = 1400

func NewPacket(reliable bool, length int) *Packet {

	p := Packet{}

	p.reliable = reliable

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
	p.reliable = reliable

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

func (p *Packet) formatDataPacket(resend bool) int {
	p.retransmit = resend

	var headerlen int

	if p.reliable {
		headerlen = 16
	} else {
		headerlen = 8
	}

	/* mod 4 */

	len := (p.datalen + headerlen + 3) & 0xfffc
	start := p.offset - headerlen

	buff := p.buff

	buff[start+1] = byte(p.scope)

	if resend {
		buff[start] |= R_DATA_PT

		intToByte(int(p.sender.getID()), buff, start+4)
		intToByte(int(p.source.getID()), buff, start+8)
	} else {
		buff[start] = (byte)(VersionNumber << 6)

		/* fill the length field */

		shortToByte(len, buff, start+2)

		intToByte(int(p.source.getID()), buff, start+4)

		if p.reliable {
			timestamp := ntp32(nowMillis())

			intToByte(timestamp, buff, start+8)
			intToByte(int(p.seqno), buff, start+12)
		} else {
			buff[start] |= U_DATA_PT
		}

		/* padding */

		pad := len - (p.datalen + headerlen)

		if pad > 0 {
			buff[start] |= byte(padBit)

			for i := start + len - 2; i > (start + len - pad); i-- {
				buff[i] = 0
			}

			buff[start+len-1] = byte(pad)
		}
	}

	return len
}

func (p *Packet) appendNackReply(ev *lossEvent, whoami *sender, firstReply int, bitmReply uint32) {
	start := p.offset

	offset := p.offset
	buff := p.buff

	buff[offset] = (byte)((VersionNumber << 6) | R_NACK_PT)
	offset++
	buff[offset] = byte(ev.scope)
	offset += 3

	intToByte(int(whoami.getID()), buff, offset)

	offset += 4

	intToByte(int(ev.reporter.getID()), buff, offset)

	offset += 4

	intToByte(ev.timestamp, buff, offset)

	offset += 4

	/*
	 * expressed in units of 1/65536 seconds (1/0x10000).
	 */
	delay := int(millis(time.Now().Sub(ev.rcvSendTime)))

	delay = millisToFixedPoint32(delay)

	intToByte(delay, buff, offset)

	offset += 4

	intToByte(int(ev.source.getID()), buff, offset)

	offset += 4

	intToByte(firstReply, buff, offset)

	offset += 4

	intToByte(int(bitmReply), buff, offset)

	offset += 4

	len := offset - start

	shortToByte(len, buff, start+2)

	p.offset = offset
}

func (p *Packet) appendNack(ev *lossEvent) {
	start := p.offset

	buff := p.buff
	offset := p.offset

	buff[offset] = (byte)((VersionNumber << 6) | NACK_PT)
	offset++
	buff[offset] = byte(ev.scope)
	offset += 3

	intToByte(int(ev.reporter.getID()), buff, offset)

	offset += 4

	intToByte(ntp32(nowMillis()), buff, offset)

	offset += 4

	intToByte(int(ev.source.getID()), buff, offset)

	offset += 4

	intToByte(int(ev.low), buff, offset)

	offset += 4

	intToByte(int(ev.bitmask), buff, offset)

	offset += 4

	len := offset - start

	/* fill the length field */

	shortToByte(len, buff, start+2)

	p.offset = offset

}
