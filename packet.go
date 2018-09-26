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

func (packet *Packet) GetDataLength() int {
	return packet.len
}
func (packet *Packet) GetMaxDataLength() int {
	return packet.maxDataLen
}

func (packet *Packet) GetData() []byte {
	return packet.buff[packet.offset : packet.offset+packet.datalen]
}

func (packet *Packet) appendSenderReport(whoami *sender) {
}

func (packet *Packet) appendRRSelection(whoami *sender, prob int, interval int) {
}

func (p *Packet) appendReceiverReport(sender *sender, whoami *sender) {
	start := p.offset

	offset := p.offset
	buff := p.buff

	buff[offset] = (byte)((VersionNumber << 6) | RR_PT)
	offset++
	buff[offset] = byte(p.scope)
	offset += 3

	intToByte(whoami.getID(), buff, offset)

	offset += 4

	intToByte(sender.getID(), buff, offset)

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

	buff[start+2] = (byte)((len >> 8) & 0xff)
	buff[start+3] = (byte)(len & 0xff)

	p.offset = offset

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
