package lrmp

type packetCache struct {
	buffer []*Packet
	mask   int
}

/**
 * constructs an LrmpPacketCache. The cache size should be multiple of
 * two (2^n).
 */
func newPacketCache(size int) packetCache {

	mask := 0

	for mask = 1; mask < size; {
		mask = mask << 1
	}

	buffer := make([]*Packet, mask)

	mask--

	pc := packetCache{buffer: buffer, mask: mask}
	return pc
}

/**
 * returns the maximum size of the cache.
 */
func (pc *packetCache) getMaxSize() int {
	return pc.mask + 1
}

/**
 * adds the given packet to the queue.
 */
func (pc *packetCache) addPacket(packet *Packet) {
	i := int(packet.seqno & int64(pc.mask))
	pc.buffer[i] = packet
}

/**
 * contains the packet.
 */
func (pc *packetCache) containPacket(seqno int64) bool {
	i := seqno & int64(pc.mask)

	if pc.buffer[i] != nil {
		return pc.buffer[i].seqno == seqno
	}
	return false
}

/**
 * gets the packet corresponding to the given seqno.
 */
func (pc *packetCache) getPacket(seqno int64) *Packet {
	i := seqno & int64(pc.mask)

	if pc.buffer[i] != nil && pc.buffer[i].seqno == seqno {
		return pc.buffer[i]
	}
	return nil
}

/**
 * remove the given packet from the queue.
 */
func (pc *packetCache) removePacket(obj *Packet) {
	i := obj.seqno & int64(pc.mask)

	if pc.buffer[i] != nil && pc.buffer[i].seqno == obj.seqno {
		pc.buffer[i] = nil
	}
}

/**
 * remove the packet with the given seqno from the queue.
 */
func (pc *packetCache) removeBySeqNo(seqno int64) {
	i := seqno & int64(pc.mask)

	if pc.buffer[i] != nil && pc.buffer[i].seqno == seqno {
		pc.buffer[i] = nil
	}
}

/**
 * remove the packet with the given seqno from the queue.
 */
func (pc *packetCache) clear() {
	for i := 0; i < len(pc.buffer); i++ {
		pc.buffer[i] = nil
	}
}
