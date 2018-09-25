package lrmp

import "container/list"

type packetQueue struct {
	list.List
}

func (pq packetQueue) isEmpty() bool {
	return pq.Len() == 0
}

/**
 * adds the given packet to the queue.
 */
func (pq packetQueue) enqueue(p *Packet) {
	pq.PushFront(p)
}

func (pq packetQueue) contains(p *Packet) bool {
	for next := pq.Front(); next != nil; next = next.Next() {
		if next.Value.(*Packet) == p {
			return true
		}
	}
	return false
}

func (pq packetQueue) dequeue() *Packet {
	front := pq.Front()
	if front == nil {
		return nil
	}
	pq.Remove(front)
	return front.Value.(*Packet)
}

/**
* remove the packet with the given seqno from the queue.
 */
func (pq packetQueue) remove(s *sender, seqno int64, scope int) {
	for next := pq.Front(); next != nil; next = next.Next() {
		p := next.Value.(*Packet)
		if p.sender.getID() == s.id && p.seqno == seqno && p.scope == scope {
			pq.Remove(next)
			break
		}
	}
}

func (pq packetQueue) cancel(s *sender, id int, scope int) {
	for next := pq.Front(); next != nil; next = next.Next() {
		p := next.Value.(*Packet)
		if p.sender.getID() == s.id && p.retransmitID == id && p.scope == scope {
			pq.Remove(next)
			break
		}
	}
}
